# PoC Plan: sx - AI Skills Package Manager

## Project Classification
**Type**: `llm-app`  
**Category**: AI tooling and governance  
**Infrastructure Profile**: Standard container deployment with persistent storage  

## Executive Summary

sx is a Go-based CLI package manager designed to solve a critical enterprise AI challenge: how to standardize, version, and distribute AI skills, MCP configurations, and commands across development teams. This PoC will validate sx as a containerized service on OpenShift AI, demonstrating centralized AI asset management and its integration potential with Red Hat AI Hub workflows.

## PoC Objectives

### Primary Goals
1. **Containerize and deploy sx** as a service on OpenShift AI with persistent vault storage
2. **Validate AI skill distribution workflow** including skill publishing, installation, and team scoping
3. **Demonstrate MCP server management** relevant to Red Hat AI's agentic strategy
4. **Test governance capabilities** including audit trails and team management

### Success Criteria
- sx CLI successfully containerized and accessible via OpenShift routes
- Persistent vault maintains state across pod restarts
- Team-scoped skill installation works correctly
- MCP server configurations can be distributed and activated
- Audit and analytics endpoints provide meaningful governance data

## Infrastructure Requirements

### Deployment Model
- **Type**: `Deployment` (long-running service)
- **Replicas**: 1 (single instance for PoC)
- **Resource Profile**: Small (CPU: 500m, Memory: 1Gi)

### Storage
- **Persistent Volume**: Required for vault storage
- **Size**: 5Gi
- **Access Mode**: ReadWriteOnce
- **Mount Path**: `/data/vault`

### Networking
- **Service Port**: 8080 (HTTP API for vault operations)
- **Route**: Required for external access
- **Internal**: Cluster-internal service for potential AI Hub integration

### Dependencies
- No external dependencies required for basic PoC
- Optional: Git repository for vault backend (can use local vault)
- Optional: Database for advanced analytics (use local files for PoC)

### Environment Variables
```yaml
SX_VAULT_TYPE: "path"
SX_VAULT_PATH: "/data/vault"
SX_SERVER_MODE: "true"
SX_HTTP_PORT: "8080"
SX_LOG_LEVEL: "info"
```

## PoC Scenarios

### Scenario 1: Basic Skill Management
**Description**: Test core functionality of adding, versioning, and installing AI skills  
**Steps**:
1. Access sx CLI via container shell
2. Initialize a new vault at `/data/vault`
3. Add a sample AI skill (coding assistant prompt)
4. Install skill with organization scope
5. Verify skill persistence across pod restart

**Expected Result**: Skill successfully stored and retrieved, demonstrating basic vault functionality

**Validation Method**: CLI commands with output verification
```bash
sx init --type path --path /data/vault
sx add sample-skill
sx install sample-skill --org
sx list
```

### Scenario 2: Team-Scoped Distribution
**Description**: Validate team management and scoped skill installation  
**Steps**:
1. Create team definitions in vault manifest
2. Add skills with different scope targets (org, team, user)
3. Simulate different user contexts
4. Verify scope resolution works correctly

**Expected Result**: Skills are properly scoped and only accessible to intended recipients

**Validation Method**: Mock different user identities and verify `sx install --dry-run` output

### Scenario 3: MCP Server Configuration
**Description**: Test MCP server definition management and distribution  
**Steps**:
1. Add MCP server configuration to vault
2. Install MCP config with repository scope
3. Export resolved configuration for client consumption
4. Verify configuration format compatibility

**Expected Result**: MCP configurations properly stored and exportable in standard format

**Validation Method**: Configuration file validation and format checking

### Scenario 4: Audit and Analytics
**Description**: Verify governance capabilities work in containerized environment  
**Steps**:
1. Perform various vault operations (add, install, remove)
2. Check audit log generation
3. Query usage analytics
4. Verify log persistence and structure

**Expected Result**: Complete audit trail with structured data for governance reporting

**Validation Method**: Examine audit logs and analytics output

### Scenario 5: HTTP API Access
**Description**: Test sx server mode for programmatic access  
**Steps**:
1. Start sx in server mode on port 8080
2. Access vault operations via HTTP API
3. Test skill installation through API calls
4. Verify API response formats

**Expected Result**: REST API provides full vault functionality for integration scenarios

**Validation Method**: HTTP requests with curl/wget and response validation

## Technical Implementation Notes

### Containerization Strategy
- Use UBI Go base image for Red Hat compatibility
- Multi-stage build: compile in builder, run in minimal runtime
- Single binary deployment with clear entrypoint
- Non-root user (1001) for security compliance

### Persistence Strategy
- Mount PVC at `/data/vault` for vault storage
- Use sx's native path vault type for simplicity
- Ensure proper permissions for non-root user
- Backup/restore capability via PVC snapshots

### Service Exposure
- ClusterIP service for internal access
- OpenShift Route for external CLI/API access
- Consider service mesh integration for enterprise scenarios

### Integration Points
- Future integration with Red Hat AI Hub for skill discovery
- Potential OIDC integration for enterprise authentication
- Event streaming to enterprise analytics platforms

## Risk Assessment

### Low Risk
- **Containerization**: Go binary is straightforward to containerize
- **Persistence**: Standard PVC patterns, well-understood
- **Basic functionality**: CLI tool with clear behavior

### Medium Risk
- **Server mode stability**: HTTP API mode may need additional testing
- **Concurrent access**: Single-instance deployment may have limitations
- **Authentication**: No built-in auth, relies on network-level security

### Mitigation Strategies
- Start with CLI-only deployment if server mode issues arise
- Use namespace-level RBAC for access control
- Document scaling considerations for production deployment

## Expected Outcomes

### Technical Validation
- sx runs successfully in OpenShift container environment
- Vault operations persist across pod lifecycle
- Team and scope management works as designed
- MCP configuration distribution is functional

### Strategic Validation
- Demonstrates Red Hat AI's comprehensive tooling approach
- Validates governance capabilities for enterprise AI adoption
- Shows integration potential with AI Hub and other Red Hat AI components
- Provides foundation for AI skill marketplace concepts

### Integration Readiness
- Container images suitable for production deployment
- API surface ready for upstream integration
- Governance data suitable for enterprise reporting
- Architecture compatible with Red Hat AI platform patterns

This PoC will demonstrate sx's viability as a core component in Red Hat AI's developer experience and governance story, providing concrete evidence of how AI skill management can be operationalized at enterprise scale.