# RDS Clusters

RDS Clusters provide a collection of database instances designed for high availability and scalability.

## Cluster Management
- **Primary Instances**: The main read-write instance in the cluster.
- **Reader Instances**: Read-only replicas that distribute the read load.
- **Failover Mechanisms**: Automated promotion of a reader instance if the primary fails.
- **Scaling Policies**: Automatically add or remove reader instances based on traffic.
