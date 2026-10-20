package config

// ChangeWorkspaceConfig registers trusted sources for the additive change workflow.
type ChangeWorkspaceConfig struct {
	Enabled          bool                  `yaml:"enabled" json:"enabled"`
	Projects         []ChangeProjectConfig `yaml:"projects" json:"-"`
	TaskFlowEndpoint string                `yaml:"taskflow_endpoint" json:"-"`
	TaskFlowTokenEnv string                `yaml:"taskflow_token_env" json:"-"`
}

// ChangeProjectConfig is operator-maintained; requests cannot supply paths or commands.
type ChangeProjectConfig struct {
	ID                     string   `yaml:"id" json:"id"`
	Name                   string   `yaml:"name" json:"name"`
	TenantID               uint64   `yaml:"tenant_id" json:"-"`
	RepositoryPath         string   `yaml:"repository_path" json:"-"`
	BusinessPackID         string   `yaml:"business_pack_id" json:"business_pack_id"`
	BusinessPackRevision   string   `yaml:"business_pack_revision" json:"business_pack_revision"`
	CheckProfileID         string   `yaml:"check_profile_id" json:"check_profile_id"`
	CheckProfileRevision   string   `yaml:"check_profile_revision" json:"check_profile_revision"`
	CheckIDs               []string `yaml:"check_ids" json:"check_ids"`
	RequiredReferencePaths []string `yaml:"required_reference_paths" json:"required_reference_paths"`
}
