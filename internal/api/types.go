package api

import "time"

type APIResponse[T any] struct {
	Success bool              `json:"success"`
	Message string            `json:"message"`
	Data    T                 `json:"data"`
	Meta    *PaginationMeta   `json:"meta,omitempty"`
	Errors  map[string][]string `json:"errors,omitempty"`
}

type PaginationMeta struct {
	CurrentPage int `json:"current_page"`
	PerPage     int `json:"per_page"`
	Total       int `json:"total"`
	LastPage    int `json:"last_page"`
}

type TwoFactorChallengeRequest struct {
	Code         string `json:"code,omitempty"`
	RecoveryCode string `json:"recovery_code,omitempty"`
}

type UserResponse struct {
	ID               string        `json:"id"`
	Name             string        `json:"name"`
	Email            string        `json:"email"`
	ProfilePhotoURL  string        `json:"profile_photo_url"`
	CurrentTeamID    *string       `json:"current_team_id"`
	CurrentTeam      *TeamResponse `json:"current_team"`
	Timezone         string        `json:"timezone"`
	Onboarded        bool          `json:"onboarded"`
	TwoFactorEnabled bool          `json:"two_factor_enabled"`
	CreatedAt        time.Time     `json:"created_at"`
}

type TeamResponse struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	UserID       string    `json:"user_id"`
	PersonalTeam bool      `json:"personal_team"`
	ImageURL     string    `json:"image_url"`
	IsSubscribed bool      `json:"is_subscribed"`
	IsOwner      bool      `json:"is_owner"`
	CreatedAt    time.Time `json:"created_at"`
}

type TeamDetailResponse struct {
	TeamResponse
	Owner       *UserResponse        `json:"owner,omitempty"`
	Members     []TeamMemberResponse `json:"members,omitempty"`
	Permissions *TeamPermissions     `json:"permissions,omitempty"`
}

type TeamMemberResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	JoinedAt  time.Time `json:"joined_at"`
	AvatarURL string    `json:"avatar_url"`
}

type TeamPermissions struct {
	CanUpdateTeam        bool `json:"can_update_team"`
	CanDeleteTeam        bool `json:"can_delete_team"`
	CanAddTeamMembers    bool `json:"can_add_team_members"`
	CanUpdateTeamMembers bool `json:"can_update_team_members"`
	CanRemoveTeamMembers bool `json:"can_remove_team_members"`
}

type ServerResponse struct {
	ID                    string   `json:"id"`
	TeamID                string   `json:"team_id"`
	Name                  string   `json:"name"`
	Description           string   `json:"description,omitempty"`
	Provider              string   `json:"provider"`
	ProviderLabel         string   `json:"provider_label"`
	Type                  string   `json:"type"`
	TypeLabel             string   `json:"type_label"`
	Connected             bool     `json:"connected"`
	MonitoringEnabled     bool     `json:"monitoring_enabled"`
	CPUCores              *int     `json:"cpu_cores,omitempty"`
	MemoryInMB            *int     `json:"memory_in_mb,omitempty"`
	StorageInGB           *int     `json:"storage_in_gb,omitempty"`
	OperatingSystem       string   `json:"operating_system"`
	OperatingSystemLabel  string   `json:"operating_system_label"`
	Status                string   `json:"status"`
	StatusLabel           string   `json:"status_label"`
	PublicIPv4            *string  `json:"public_ipv4,omitempty"`
	Username              string   `json:"username"`
	SSHPort               int      `json:"ssh_port"`
	AutoUpdate            bool     `json:"auto_update"`
	Progress              *int     `json:"progress,omitempty"`
	ProgressStep          *string  `json:"progress_step,omitempty"`
	ProvisionedAt         *string  `json:"provisioned_at,omitempty"`
	LastConnectivityCheck *string  `json:"last_connectivity_check,omitempty"`
	ArchivedAt            *string  `json:"archived_at,omitempty"`
	CreatedAt             string   `json:"created_at"`
	UpdatedAt             string   `json:"updated_at"`
	Features              []string `json:"features,omitempty"`
	SitesCount            int      `json:"sites_count"`
	ServicesCount         int      `json:"services_count"`
	UpstreamsCount        int      `json:"upstreams_count"`
}

type MetricResponse struct {
	ID                 int     `json:"id"`
	ServerID           string  `json:"server_id"`
	Load               float64 `json:"load"`
	MemoryTotal        float64 `json:"memory_total"`
	MemoryUsed         float64 `json:"memory_used"`
	MemoryFree         float64 `json:"memory_free"`
	MemoryUsagePercent float64 `json:"memory_usage_percent"`
	DiskTotal          float64 `json:"disk_total"`
	DiskUsed           float64 `json:"disk_used"`
	DiskFree           float64 `json:"disk_free"`
	DiskUsagePercent   float64 `json:"disk_usage_percent"`
	CreatedAt          string  `json:"created_at"`
}

type SiteResponse struct {
	ID                           string              `json:"id"`
	ServerID                     string              `json:"server_id"`
	Address                      string              `json:"address"`
	Type                         string              `json:"type"`
	TLSSetting                   string              `json:"tls_setting"`
	ZeroDowntimeDeployment       bool                `json:"zero_downtime_deployment"`
	DeploymentReleasesRetention  int                 `json:"deployment_releases_retention"`
	RepositoryBranch             string              `json:"repository_branch"`
	Path                         string              `json:"path"`
	WebFolder                    string              `json:"web_folder"`
	PHPVersion                   string              `json:"php_version"`
	AutoDeployment               bool                `json:"auto_deployment"`
	URL                          string              `json:"url"`
	RepositoryURL                *string             `json:"repository_url,omitempty"`
	Status                       string              `json:"status"`
	InstalledAt                  *string             `json:"installed_at,omitempty"`
	LatestDeployment             *DeploymentResponse `json:"latest_deployment,omitempty"`
	CreatedAt                    string              `json:"created_at"`
	UpdatedAt                    string              `json:"updated_at"`
}

type DeploymentResponse struct {
	ID           string      `json:"id"`
	SiteID       string      `json:"site_id"`
	ServerID     string      `json:"server_id,omitempty"`
	UserID       *string     `json:"user_id,omitempty"`
	TaskID       *string     `json:"task_id,omitempty"`
	Status       string      `json:"status"`
	GitHash      *string     `json:"git_hash,omitempty"`
	ShortGitHash string      `json:"short_git_hash"`
	CommitData   *CommitData `json:"commit_data,omitempty"`
	IsRollback   bool        `json:"is_rollback"`
	CreatedAt    string      `json:"created_at"`
	UpdatedAt    string      `json:"updated_at"`
}

type TaskResponse struct {
	ID        string  `json:"id"`
	ServerID  string  `json:"server_id"`
	Type      string  `json:"type"`
	Status    string  `json:"status"`
	Name      string  `json:"name"`
	User      string  `json:"user"`
	Output    *string `json:"output,omitempty"`
	ExitCode  *int    `json:"exit_code,omitempty"`
	CreatedAt string  `json:"created_at"`
}

type CommitData struct {
	Message string `json:"message"`
	Author  string `json:"author"`
	URL     string `json:"url"`
}

type DashboardResponse struct {
	Servers        []DashboardServer   `json:"servers"`
	RecentActivity []DashboardActivity `json:"recent_activity"`
}

type DashboardServer struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Status     string `json:"status"`
	Provider   string `json:"provider"`
	SitesCount int    `json:"sites_count"`
}

type DashboardActivity struct {
	ID            string         `json:"id"`
	SiteName      string         `json:"site_name"`
	SiteID        string         `json:"site_id"`
	ServerID      string         `json:"server_id"`
	ServerName    string         `json:"server_name"`
	Status        string         `json:"status"`
	CreatedAt     string         `json:"created_at"`
	CommitSHA     string         `json:"commit_sha,omitempty"`
	CommitMessage string         `json:"commit_message,omitempty"`
	User          *ActivityUser  `json:"user,omitempty"`
}

type ActivityUser struct {
	Name string `json:"name"`
}

type AuthResponse struct {
	User         UserResponse `json:"user"`
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	ExpiresIn    int          `json:"expires_in"`
	TokenType    string       `json:"token_type"`
}

type CreateServerRequest struct {
	Name            string `json:"name"`
	Provider        string `json:"provider"`
	CredentialID    string `json:"credential_id"`
	Type            string `json:"type"`
	Region          string `json:"region"`
	Size            string `json:"size"`
	OperatingSystem string `json:"operating_system"`
	PHPVersion      string `json:"php_version,omitempty"`
}

type CreateServerOptions struct {
	Providers       []ProviderOption `json:"providers"`
	OperatingSystems []SelectOption  `json:"operating_systems"`
	Types           []SelectOption   `json:"types"`
	PHPVersions     []SelectOption   `json:"php_versions,omitempty"`
}

type ProviderOption struct {
	ID       string         `json:"id"`
	Name     string         `json:"name"`
	Provider string         `json:"provider"`
	Regions  []SelectOption `json:"regions"`
	Sizes    []SelectOption `json:"sizes"`
}

type SelectOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

type CreateSiteRequest struct {
	Address        string `json:"address"`
	Type           string `json:"type"`
	PHPVersion     string `json:"php_version,omitempty"`
	WebFolder      string `json:"web_folder,omitempty"`
	ZeroDowntime   bool   `json:"zero_downtime_deployment"`
}

type SwitchTeamRequest struct {
	TeamID string `json:"team_id"`
}

