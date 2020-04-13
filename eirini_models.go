package eirinistaging

const (
	//Environment Variable Names
	EnvDownloadURL                     = "DOWNLOAD_URL"
	EnvBuildpacks                      = "BUILDPACKS"
	EnvDropletUploadURL                = "DROPLET_UPLOAD_URL"
	EnvAppID                           = "APP_ID"
	EnvStagingGUID                     = "STAGING_GUID"
	EnvCompletionCallback              = "COMPLETION_CALLBACK"
	EnvCfUsername                      = "CF_USERNAME"
	EnvCfPassword                      = "CF_PASSWORD"
	EnvAPIAddress                      = "API_ADDRESS"
	EnvEiriniAddress                   = "EIRINI_ADDRESS"
	EnvCertsPath                       = "EIRINI_CERTS_PATH"
	EnvBuildpacksDir                   = "EIRINI_BUILDPACKS_DIR"
	EnvWorkspaceDir                    = "EIRINI_WORKSPACE_DIR"
	EnvOutputDropletLocation           = "EIRINI_OUTPUT_DROPLET_LOCATION"
	EnvOutputBuildArtifactsCache       = "EIRINI_OUTPUT_BUILD_ARTIFACTS_CACHE"
	EnvOutputMetadataLocation          = "EIRINI_OUTPUT_METADATA_LOCATION"
	EnvBuildArtifactsCacheDir          = "EIRINI_BUILD_ARTIFACTS_CACHE_DIR"
	EnvBuildpackCacheUploadURI         = "BUILDPACK_CACHE_UPLOAD_URI"
	EnvBuildpackCacheDownloadURI       = "BUILDPACK_CACHE_DOWNLOAD_URI"
	EnvBuildpackCacheChecksum          = "BUILDPACK_CACHE_CHECKSUM"
	EnvBuildpackCacheChecksumAlgorithm = "BUILDPACK_CACHE_CHECKSUM_ALGORITHM"

	RegisteredRoutes = "routes"

	AppBits                         = "app.zip"
	RecipeBuildPacksDir             = "/var/lib/buildpacks"
	RecipeBuildPacksName            = "recipe-buildpacks"
	RecipeWorkspaceDir              = "/recipe_workspace"
	RecipeWorkspaceName             = "recipe-workspace"
	RecipeOutputName                = "staging-output"
	RecipeOutputLocation            = "/out"
	RecipeOutputDropletLocation     = "/out/droplet.tgz"
	RecipeOutputBuildArtifactsCache = "/buildpack-cache/cache.tgz"
	RecipeOutputMetadataLocation    = "/out/result.json"
	BuildArtifactsCacheDir          = "/buildpack-cache/cache"

	CCUploaderInternalURL = "cc-uploader.service.cf.internal"

	CACertName = "internal-ca-cert"

	CCCertsMountPath = "/etc/config/certs"
	CCAPICertName    = "cc-server-crt"
	CCAPIKeyName     = "cc-server-crt-key"

	EiriniClientCert = "eirini-client-crt"
	EiriniClientKey  = "eirini-client-crt-key"
)

type Extractor interface {
	Extract(src, targetDir string) error
}
