// Package store is the implementation for managing Bytebase's own metadata in a PostgreSQL database.
package store

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/dgraph-io/ristretto"
	"google.golang.org/protobuf/encoding/protojson"

	api "github.com/bytebase/bytebase/backend/legacyapi"
)

var (
	protojsonUnmarshaler = protojson.UnmarshalOptions{
		DiscardUnknown: true,
	}
)

// Store provides database access to all raw objects.
type Store struct {
	db *DB

	userIDCache                    sync.Map         // map[int]*UserMessage
	environmentCache               sync.Map         // map[string]*EnvironmentMessage
	environmentIDCache             sync.Map         // map[int]*EnvironmentMessage
	instanceCache                  sync.Map         // map[string]*InstanceMessage
	instanceIDCache                sync.Map         // map[int]*InstanceMessage
	databaseCache                  sync.Map         // map[string]*DatabaseMessage
	databaseIDCache                sync.Map         // map[int]*DatabaseMessage
	projectCache                   sync.Map         // map[string]*ProjectMessage
	projectIDCache                 sync.Map         // map[int]*ProjectMessage
	projectPolicyCache             sync.Map         // map[string]*IAMPolicyMessage
	projectIDPolicyCache           sync.Map         // map[int]*IAMPolicyMessage
	policyCache                    sync.Map         // map[string]*PolicyMessage
	issueCache                     sync.Map         // map[int]*IssueMessage
	issueByPipelineCache           sync.Map         // map[int]*IssueMessage
	pipelineCache                  sync.Map         // map[int]*PipelineMessage
	dbSchemaCache                  *ristretto.Cache // map[int]*DBSchema
	settingCache                   sync.Map         // map[string]*SettingMessage
	idpCache                       sync.Map         // map[string]*IdentityProvider
	projectIDDeploymentConfigCache sync.Map         // map[int]*DeploymentConfigMessage
	risksCache                     sync.Map         // []*RiskMessage, use 0 as the key
	databaseGroupCache             sync.Map         // map[string]*DatabaseGroupMessage
	databaseGroupIDCache           sync.Map         // map[int]*DatabaseGroupMessage
	schemaGroupCache               sync.Map         // map[string]*SchemaGroupMessage
	// sheetStatementCache caches the statement of a sheet.
	sheetStatementCache *ristretto.Cache // map[sheetUID]sheetStatementString
	vcsIDCache          sync.Map         // map[int]*ExternalVersionControlMessage
}

// New creates a new instance of Store.
func New(db *DB) *Store {
	sheetStatementCache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1_000,
		MaxCost:     1_000_000_000, // ~1GB
		BufferItems: 64,
	})
	if err != nil {
		panic(fmt.Sprintf("failed to create sheet statement cache: %v", err))
	}
	dbsCache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 10_000,
		MaxCost:     1_000_000, // ~1MB
		BufferItems: 64,
	})
	if err != nil {
		panic(fmt.Sprintf("failed to create sheet statement cache: %v", err))
	}
	return &Store{
		db:                  db,
		sheetStatementCache: sheetStatementCache,
		dbSchemaCache:       dbsCache,
	}
}

// RefreshSwap updates the cache.
func (s *Store) RefreshSwap(refresh bool) {
	cost := int64(1_000_000)
	if refresh {
		cost = 500_000_000
	}
	s.dbSchemaCache.UpdateMaxCost(cost)
}

// Close closes underlying db.
func (s *Store) Close(ctx context.Context) error {
	return s.db.Close(ctx)
}

func getInstanceCacheKey(instanceID string) string {
	return instanceID
}

func getPolicyCacheKey(resourceType api.PolicyResourceType, resourceUID int, policyType api.PolicyType) string {
	return fmt.Sprintf("policies/%s/%d/%s", resourceType, resourceUID, policyType)
}

func getDatabaseCacheKey(instanceID, databaseName string) string {
	return fmt.Sprintf("%s/%s", instanceID, databaseName)
}

func getDatabaseGroupCacheKey(projectUID int, databaseGroupResourceID string) string {
	return fmt.Sprintf("%d/%s", projectUID, databaseGroupResourceID)
}

func getSchemaGroupCacheKey(databaseGroupUID int64, schemaGroupResourceID string) string {
	return fmt.Sprintf("%d/%s", databaseGroupUID, schemaGroupResourceID)
}

func getPlaceholders(start int, count int) string {
	var placeholders []string
	for i := start; i < start+count; i++ {
		placeholders = append(placeholders, fmt.Sprintf("$%d", i))
	}
	return fmt.Sprintf("(%s)", strings.Join(placeholders, ","))
}
