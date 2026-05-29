package vault

import (
	"context"
	"fmt"
	"strings"

	"github.com/sleuth-io/sx/internal/lockfile"
	"github.com/sleuth-io/sx/internal/manifest"
	"github.com/sleuth-io/sx/internal/metadata"
	"github.com/sleuth-io/sx/internal/utils"
	"github.com/sleuth-io/sx/internal/version"
)

func manifestAssetVersions(vaultRoot, name string) ([]string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return []string{}, nil
	}
	m, ok, err := manifest.Load(vaultRoot)
	if err != nil {
		return nil, err
	}
	if !ok {
		return []string{}, nil
	}
	versions := []string{}
	for _, asset := range m.Assets {
		if asset.Name != name || strings.TrimSpace(asset.Version) == "" {
			continue
		}
		versions = append(versions, asset.Version)
	}
	return version.Sort(versions), nil
}

func findAssetVersionInManifest(vaultRoot, name, version string) (*lockfile.Asset, bool, error) {
	name = strings.TrimSpace(name)
	version = strings.TrimSpace(version)
	if name == "" || version == "" {
		return nil, false, nil
	}
	m, ok, err := manifest.Load(vaultRoot)
	if err != nil {
		return nil, false, err
	}
	if !ok {
		return nil, false, nil
	}
	var found *lockfile.Asset
	for _, asset := range m.Assets {
		if asset.Name == name && asset.Version == version {
			out := manifestAssetToLockfile(asset)
			if found != nil {
				return nil, false, fmt.Errorf("duplicate manifest asset %q version %q", name, version)
			}
			found = &out
		}
	}
	if found != nil {
		return found, true, nil
	}
	return nil, false, nil
}

func metadataFromAssetZip(ctx context.Context, repo Vault, asset *lockfile.Asset) (*metadata.Metadata, error) {
	data, err := repo.GetAsset(ctx, asset)
	if err != nil {
		return nil, err
	}
	raw, err := utils.ReadZipFile(data, "metadata.toml")
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata.toml from asset zip: %w", err)
	}
	return metadata.Parse(raw)
}
