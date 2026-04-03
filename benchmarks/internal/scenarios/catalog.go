package scenarios

type Identity struct {
	ID       string `json:"id"`
	Category string `json:"category"`
}

type InventoryRecord struct {
	Identity
	TaskShape     string   `json:"task_shape"`
	Summary       string   `json:"summary"`
	TruthSurfaces []string `json:"truth_surfaces"`
}

func NativeReferenceInventory() []InventoryRecord {
	return []InventoryRecord{
		{
			Identity: Identity{
				ID:       "relative_navigation_session",
				Category: "relative_path_navigation",
			},
			TaskShape: "session_local_navigation",
			Summary:   "Exercises session-scoped cwd changes, relative path traversal, and low-noise path feedback inside the default workspace.",
			TruthSurfaces: []string{
				"virtual_cwd",
				"relative_path_resolution",
				"execution_trace",
			},
		},
		{
			Identity: Identity{
				ID:       "inspect_edit_write_loop",
				Category: "file_inspect_edit_write_loops",
			},
			TaskShape: "inspect_edit_write_patch_workflow",
			Summary:   "Exercises the common read-edit-write loop for file inspection, mutation, and patch-style iteration.",
			TruthSurfaces: []string{
				"file_mutation",
				"execution_trace",
				"session_continuity",
			},
		},
		{
			Identity: Identity{
				ID:       "mount_boundary_relative_path",
				Category: "mount_synthetic_capability_boundaries",
			},
			TaskShape: "mount_boundary_and_denial",
			Summary:   "Exercises mount-backed and synthetic capability boundaries, including relative access and explicit denial semantics.",
			TruthSurfaces: []string{
				"path_capabilities",
				"denied_paths",
				"relative_path_resolution",
			},
		},
		{
			Identity: Identity{
				ID:       "command_namespace_consistency",
				Category: "command_namespace_consistency",
			},
			TaskShape: "command_introspection",
			Summary:   "Exercises builtin namespace discovery and command-manual consistency rather than repo-task execution pressure.",
			TruthSurfaces: []string{
				"builtin_namespace",
				"manual_summary",
			},
		},
		{
			Identity: Identity{
				ID:       "trace_consumable_planning",
				Category: "trace_consumable_planning",
			},
			TaskShape: "trace_to_planning_feedback",
			Summary:   "Exercises whether structured trace and result surfaces are legible enough to drive downstream planning behavior.",
			TruthSurfaces: []string{
				"execution_trace",
				"structured_result_feedback",
			},
		},
		{
			Identity: Identity{
				ID:       "adapter_projection_memory_lifecycle",
				Category: "adapter_backed_projection_validation",
			},
			TaskShape: "adapter_projection_lifecycle",
			Summary:   "Exercises rich adapter-backed projection, managed memory, skills selection, audit, metrics, denials, and workflow state.",
			TruthSurfaces: []string{
				"projection_indexes",
				"managed_memory",
				"workflow_views",
				"skills_selection",
				"control_plane_audit",
				"projection_metrics",
				"denial_surfaces",
			},
		},
		{
			Identity: Identity{
				ID:       "adapter_composition_evolution_stress",
				Category: "adapter_composition_evolution_truth",
			},
			TaskShape: "multi_step_adapter_evolution",
			Summary:   "Exercises multi-step composition where projection refresh, control-plane mutation, denials, and resume must remain aligned together.",
			TruthSurfaces: []string{
				"projection_indexes",
				"freshness_materialization",
				"control_plane_audit",
				"projection_metrics",
				"denial_surfaces",
				"checkpoint_resume",
			},
		},
		{
			Identity: Identity{
				ID:       "resource_set_adapter_seam",
				Category: "adapter_second_seam_validation",
			},
			TaskShape: "second_adapter_minimal_seam",
			Summary:   "Exercises the smaller resourceset adapter so seam validation does not silently overfit to the richer reference adapter.",
			TruthSurfaces: []string{
				"resource_projection",
				"managed_memory",
				"denial_surfaces",
				"checkpoint_resume",
			},
		},
		{
			Identity: Identity{
				ID:       "cancel_timeout_interruptions",
				Category: "cancel_timeout_interruption",
			},
			TaskShape: "interruptibility_and_budget_enforcement",
			Summary:   "Exercises cancellation and timeout handling so interruption truth remains explicit to callers and traces.",
			TruthSurfaces: []string{
				"cancellation",
				"timeout",
				"execution_result",
			},
		},
	}
}

func NativeReferenceIdentities() []Identity {
	inventory := NativeReferenceInventory()
	identities := make([]Identity, 0, len(inventory))
	for _, record := range inventory {
		identities = append(identities, record.Identity)
	}
	return identities
}

func NativeReferenceIDs() []string {
	identities := NativeReferenceIdentities()
	ids := make([]string, 0, len(identities))
	for _, identity := range identities {
		ids = append(ids, identity.ID)
	}
	return ids
}

func LookupNativeReferenceIdentity(id string) (Identity, bool) {
	for _, identity := range NativeReferenceIdentities() {
		if identity.ID == id {
			return identity, true
		}
	}
	return Identity{}, false
}
