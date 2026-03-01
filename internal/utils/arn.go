package utils

import "fmt"

// GenerateDatabaseARN creates a standard ARN for a database instance
func GenerateDatabaseARN(region, accountID, databaseID string) string {
	return fmt.Sprintf("arn:serw:rds:%s:%s:db/%s", region, accountID, databaseID)
}

// GenerateVolumeARN creates a standard ARN for a storage volume
func GenerateVolumeARN(region, accountID, volumeID string) string {
	return fmt.Sprintf("arn:serw:rds:%s:%s:volume/%s", region, accountID, volumeID)
}

// GenerateSnapshotARN creates a standard ARN for a database backup snapshot
func GenerateSnapshotARN(region, accountID, snapshotID string) string {
	return fmt.Sprintf("arn:serw:rds:%s:%s:snapshot/%s", region, accountID, snapshotID)
}
