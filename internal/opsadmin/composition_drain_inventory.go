package opsadmin

type DrainDisposition string

const (
	DrainToDomain  DrainDisposition = "drain"
	DrainKeepShell DrainDisposition = "keep_shell"
)

type CompositionDrainEntry struct {
	SourceFile    string
	TargetPackage string
	BridgeFile    string
	DrainSlug     string
	Disposition   DrainDisposition
}

var CompositionDrainInventory = []CompositionDrainEntry{
	{SourceFile: "billing_bridge.go", TargetPackage: "internal/billingadmin", BridgeFile: "billing_bridge.go", DrainSlug: "drain_billing_service_to_billingadmin", Disposition: DrainKeepShell},
	{SourceFile: "outbox_bridge.go", TargetPackage: "internal/outbox", BridgeFile: "outbox_bridge.go", DrainSlug: "outbox_domain_final_cut", Disposition: DrainKeepShell},
	{SourceFile: "campaign_service_bridge.go", TargetPackage: "internal/campaign", BridgeFile: "campaign_runtime_bridge.go", DrainSlug: "controlplane_wire_only_shell", Disposition: DrainKeepShell},
	{SourceFile: "nodeadmin_bridge.go", TargetPackage: "internal/nodeadmin", BridgeFile: "nodeadmin_bridge.go", DrainSlug: "controlplane_wire_only_shell", Disposition: DrainKeepShell},
	{SourceFile: "shardadmin_bridge.go", TargetPackage: "internal/shardadmin", BridgeFile: "shardadmin_bridge.go", DrainSlug: "controlplane_wire_only_shell", Disposition: DrainKeepShell},
}

func CompositionDrainBySource(name string) *CompositionDrainEntry {
	for i := range CompositionDrainInventory {
		if CompositionDrainInventory[i].SourceFile == name {
			return &CompositionDrainInventory[i]
		}
	}
	return nil
}
