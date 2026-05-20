package commands

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/sleuth-io/sx/internal/assets"
	"github.com/sleuth-io/sx/internal/clients"
	"github.com/sleuth-io/sx/internal/config"
	"github.com/sleuth-io/sx/internal/lockfile"
	"github.com/sleuth-io/sx/internal/logger"
	"github.com/sleuth-io/sx/internal/mgmt"
	"github.com/sleuth-io/sx/internal/scope"
	"github.com/sleuth-io/sx/internal/ui"
	"github.com/sleuth-io/sx/internal/ui/components"
	vaultpkg "github.com/sleuth-io/sx/internal/vault"
)

// profileLockFile pairs a fetched lock file with the profile and vault
// it came from. Used by the multi-active install flow to route asset
// downloads back to the originating vault and to attribute conflicts.
type profileLockFile struct {
	ProfileName string
	Config      *config.Config
	Vault       vaultpkg.Vault
	LockFile    *lockfile.LockFile
	FetchErr    error
}

// loadActiveProfilesAndLockFiles is the top-level helper for runInstall's
// per-profile bootstrap. It loads every active profile, fetches each
// vault's lock file, applies the partial-failure policy, and returns
// the data the rest of the install flow needs. The done bool is true
// when runInstall should return nil (e.g. all profiles report no lock
// file yet — fresh setup).
func loadActiveProfilesAndLockFiles(
	ctx context.Context,
	status *components.Status,
	styledOut *ui.Output,
) (profileLocks []profileLockFile, mpc *config.MultiProfileConfig, primaryCfg *config.Config, cfg *config.Config, done bool, err error) {
	activeConfigs, mpc, err := config.LoadActive()
	if err != nil {
		return nil, nil, nil, nil, false, fmt.Errorf("failed to load configuration: %w\nRun 'sx init' to configure", err)
	}
	if len(activeConfigs) == 0 {
		return nil, nil, nil, nil, false, errors.New("no active profiles configured — run 'sx profile activate <name>'")
	}
	primaryCfg = activeConfigs[0]
	if validateErr := primaryCfg.Validate(); validateErr != nil {
		return nil, nil, nil, nil, false, fmt.Errorf("invalid configuration for profile %s: %w", primaryCfg.ProfileName, validateErr)
	}
	mgmt.SetIdentityOverride(primaryCfg.Identity)

	profileLocks = loadActiveLockFiles(ctx, activeConfigs, status)
	if !reportFetchErrors(profileLocks, styledOut) {
		// All profiles failed. A pristine "no lock file yet" outcome is
		// the new-user case (every profile reports ErrLockFileNotFound).
		// Anything else means the warnings just printed by
		// reportFetchErrors are the diagnostic; bail with a non-zero
		// status but skip re-rendering the underlying errors.
		for _, pl := range profileLocks {
			if pl.FetchErr == nil {
				continue
			}
			if !errors.Is(pl.FetchErr, vaultpkg.ErrLockFileNotFound) {
				return nil, nil, nil, nil, false, errors.New("no active profile produced a lock file (see warnings above)")
			}
		}
		styledOut.Info("No assets installed yet.")
		styledOut.Muted("Add skills with 'sx add' or browse skills.sh with 'sx add --browse'.")
		return nil, nil, nil, nil, true, nil
	}

	// Primary config (used for downstream env detection) comes from the
	// first profile that successfully fetched a lock file. Asset-level
	// vault routing happens later via mergeApplicableAssets.
	cfg = primaryCfg
	for _, pl := range profileLocks {
		if pl.LockFile != nil {
			cfg = pl.Config
			break
		}
	}
	return profileLocks, mpc, primaryCfg, cfg, false, nil
}

// loadActiveLockFiles fetches lock files for every active profile,
// honoring per-profile identity overrides so team/user scope resolution
// happens against the right email. Individual fetch failures are
// captured per-profile rather than failing the whole call so partial
// installs can proceed.
func loadActiveLockFiles(ctx context.Context, configs []*config.Config, status *components.Status) []profileLockFile {
	results := make([]profileLockFile, 0, len(configs))
	for _, cfg := range configs {
		entry := profileLockFile{ProfileName: cfg.ProfileName, Config: cfg}
		if err := cfg.Validate(); err != nil {
			entry.FetchErr = fmt.Errorf("invalid configuration for profile %s: %w", cfg.ProfileName, err)
			results = append(results, entry)
			continue
		}
		// Switch identity context to this profile before any vault op that
		// resolves the caller's actor.
		mgmt.SetIdentityOverride(cfg.Identity)
		mgmt.SetAuditProfileTag(cfg.ProfileName)
		vault, err := vaultpkg.NewFromConfig(cfg)
		if err != nil {
			entry.FetchErr = fmt.Errorf("failed to create vault for profile %s: %w", cfg.ProfileName, err)
			results = append(results, entry)
			continue
		}
		entry.Vault = vault
		lf, err := fetchLockFileWithCache(ctx, vault, cfg, status)
		if err != nil {
			entry.FetchErr = err
		} else {
			entry.LockFile = lf
		}
		results = append(results, entry)
	}
	return results
}

// assetConflict records that two or more active profiles publish an
// asset with the same name. Winner is the profile whose copy is being
// installed; Shadowed is the rest.
type assetConflict struct {
	AssetName string
	Winner    string
	Shadowed  []string
}

// mergeApplicableAssets runs scope filtering + dependency resolution per
// profile, then folds the results into a single ordered list. The
// caller controls precedence by ordering profileLocks (default-first
// for the persisted case, user-specified for --profile overrides). On
// name collision the first encountered wins; later occurrences are
// reported via conflicts so reportConflicts can decide how loudly to
// surface them.
//
// Per-profile identity was already applied during lock fetch in
// loadActiveLockFiles; the resolver only reads lockfile bytes, so it
// doesn't touch the identity override here.
func mergeApplicableAssets(
	profileLocks []profileLockFile,
	targetClients []clients.Client,
	matcherScope *scope.Matcher,
) (sortedAssets []*lockfile.Asset, assetVault map[string]vaultpkg.Vault, conflicts []assetConflict, err error) {
	assetVault = make(map[string]vaultpkg.Vault)
	assetOrigin := make(map[string]string)
	conflictByName := make(map[string]*assetConflict)

	for _, pl := range profileLocks {
		if pl.LockFile == nil {
			continue
		}
		applicable := filterAssetsByScope(pl.LockFile, targetClients, matcherScope)
		sorted, resolveErr := resolveAssetDependencies(pl.LockFile, applicable)
		if resolveErr != nil {
			return nil, nil, nil, fmt.Errorf("dependency resolution for profile %s: %w", pl.ProfileName, resolveErr)
		}
		for _, asset := range sorted {
			if existing, taken := assetOrigin[asset.Name]; taken {
				rec, ok := conflictByName[asset.Name]
				if !ok {
					rec = &assetConflict{AssetName: asset.Name, Winner: existing}
					conflictByName[asset.Name] = rec
				}
				rec.Shadowed = append(rec.Shadowed, pl.ProfileName)
				continue
			}
			sortedAssets = append(sortedAssets, asset)
			assetVault[asset.Name] = pl.Vault
			assetOrigin[asset.Name] = pl.ProfileName
		}
	}

	if len(conflictByName) > 0 {
		names := make([]string, 0, len(conflictByName))
		for n := range conflictByName {
			names = append(names, n)
		}
		sort.Strings(names)
		for _, n := range names {
			conflicts = append(conflicts, *conflictByName[n])
		}
	}
	return sortedAssets, assetVault, conflicts, nil
}

// reportFetchErrors surfaces per-profile lock file fetch failures as
// warnings. Returns true if at least one profile fetched successfully.
func reportFetchErrors(profileLocks []profileLockFile, styledOut *ui.Output) bool {
	log := logger.Get()
	successCount := 0
	for _, pl := range profileLocks {
		if pl.LockFile != nil {
			successCount++
			continue
		}
		if pl.FetchErr == nil || errors.Is(pl.FetchErr, vaultpkg.ErrLockFileNotFound) {
			continue
		}
		styledOut.Warning(fmt.Sprintf("profile %s: %v", pl.ProfileName, pl.FetchErr))
		log.Error("profile lockfile fetch failed", "profile", pl.ProfileName, "error", pl.FetchErr)
	}
	return successCount > 0
}

// reportConflicts emits a per-shadowed-asset notice. The agreed policy
// is "default wins silently, otherwise first-active wins with a
// warning" — so we suppress the loud warning only when the winner is
// actually the persisted default profile.
func reportConflicts(conflicts []assetConflict, defaultProfile string, styledOut *ui.Output) {
	log := logger.Get()
	for _, c := range conflicts {
		shadowed := c.Shadowed
		msg := fmt.Sprintf("asset %s: kept from %s, shadowed in %v", c.AssetName, c.Winner, shadowed)
		log.Warn("asset conflict between profiles", "asset", c.AssetName, "winner", c.Winner, "shadowed", shadowed)
		if defaultProfile != "" && c.Winner == defaultProfile {
			styledOut.Muted(msg)
			continue
		}
		styledOut.Warning(msg)
	}
}

// downloadAssetsMultiVault downloads each asset from the vault its
// origin profile points at, then aggregates the results into the same
// shape as the single-vault downloader so the rest of the install flow
// is unchanged.
func downloadAssetsMultiVault(
	ctx context.Context,
	assetsToInstall []*lockfile.Asset,
	assetVault map[string]vaultpkg.Vault,
	status *components.Status,
	styledOut *ui.Output,
) (*downloadAssetsResult, error) {
	if len(assetsToInstall) == 0 {
		return &downloadAssetsResult{}, nil
	}

	// Group assets by their backing vault so we issue one batched fetch
	// per vault (preserving the existing per-vault concurrency limit).
	type group struct {
		vault  vaultpkg.Vault
		assets []*lockfile.Asset
	}
	groups := make(map[vaultpkg.Vault]*group)
	for _, asset := range assetsToInstall {
		v, ok := assetVault[asset.Name]
		if !ok || v == nil {
			return nil, fmt.Errorf("no vault routing for asset %s", asset.Name)
		}
		g, exists := groups[v]
		if !exists {
			g = &group{vault: v}
			groups[v] = g
		}
		g.assets = append(g.assets, asset)
	}

	status.Start(fmt.Sprintf("Downloading %d assets", len(assetsToInstall)))

	var merged []assets.DownloadResult
	for _, g := range groups {
		fetcher := assets.NewAssetFetcher(g.vault)
		results, err := fetcher.FetchAssets(ctx, g.assets, 10)
		if err != nil {
			status.Clear()
			return nil, fmt.Errorf("failed to fetch assets: %w", err)
		}
		merged = append(merged, results...)
	}

	result := processDownloadResults(merged, styledOut)
	status.Clear()

	if len(result.Downloads) == 0 {
		styledOut.Error("No assets downloaded successfully")
		return nil, errors.New("no assets downloaded successfully")
	}

	return result, nil
}
