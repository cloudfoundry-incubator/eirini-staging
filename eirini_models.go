package eirinistaging

const (
	//Environment Variable Names
	EnvDownloadURL               = "DOWNLOAD_URL"
	EnvBuildpacks                = "BUILDPACKS"
	EnvDropletUploadURL          = "DROPLET_UPLOAD_URL"
	EnvAppID                     = "APP_ID"
	EnvStagingGUID               = "STAGING_GUID"
	EnvCompletionCallback        = "COMPLETION_CALLBACK"
	EnvCfUsername                = "CF_USERNAME"
	EnvCfPassword                = "CF_PASSWORD"
	EnvAPIAddress                = "API_ADDRESS"
	EnvEiriniAddress             = "EIRINI_ADDRESS"
	EnvCertsPath                 = "EIRINI_CERTS_PATH"
	EnvBuildpacksDir             = "EIRINI_BUILDPACKS_DIR"
	EnvWorkspaceDir              = "EIRINI_WORKSPACE_DIR"
	EnvOutputDropletLocation     = "EIRINI_OUTPUT_DROPLET_LOCATION"
	EnvOutputBuildArtifactsCache = "EIRINI_OUTPUT_BUILD_ARTIFACTS_CACHE"
	EnvOutputMetadataLocation    = "EIRINI_OUTPUT_METADATA_LOCATION"
	EnvPacksBuilderPath          = "EIRINI_PACKS_BUILDER_PATH"

	RegisteredRoutes = "routes"

	AppBits                         = "app.zip"
	RecipeBuildPacksDir             = "/var/lib/buildpacks"
	RecipeBuildPacksName            = "recipe-buildpacks"
	RecipeWorkspaceDir              = "/recipe_workspace"
	RecipeWorkspaceName             = "recipe-workspace"
	RecipeOutputName                = "staging-output"
	RecipeOutputLocation            = "/out"
	RecipeOutputDropletLocation     = "/out/droplet.tgz"
	RecipeOutputBuildArtifactsCache = "/cache/cache.tgz"
	RecipeOutputMetadataLocation    = "/out/result.json"
	RecipePacksBuilderPath          = "/packs/builder"

	CCUploaderInternalURL = "cc-uploader.service.cf.internal"
	CCCertsMountPath      = "/etc/config/certs"
	CCCertsVolumeName     = "cc-certs-volume"
	CCAPICertName         = "cc-server-crt"
	CCAPIKeyName          = "cc-server-crt-key"
	CCUploaderCertName    = "cc-uploader-crt"
	CCUploaderKeyName     = "cc-uploader-crt-key"
	CCInternalCACertName  = "internal-ca-cert"
)

//go:generate counterfeiter . Extractor
type Extractor interface {
	Extract(src, targetDir string) error
}
