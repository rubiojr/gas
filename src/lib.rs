use serde::Deserialize;
use std::fs;
use zed::settings::ContextServerSettings;
use zed_extension_api::{
    self as zed, serde_json, Command, ContextServerId, GithubReleaseOptions, Project, Result
};

const BINARY_NAME: &str = "github-gas-server";

#[derive(Debug, Deserialize)]
struct GitHubActivitySummarizerServerSettings {
    repositories: Option<Vec<String>>,
    query_extra: Option<String>,
    from_date: Option<String>,
    author: String,
}

impl zed::Extension for GitHubGASExtension {
    fn new() -> Self {
        Self {
            cached_binary_path: None,
        }
    }

    fn context_server_command(
        &mut self,
        _context_server_id: &ContextServerId,
        _project: &Project,
    ) -> Result<Command> {
        let settings = ContextServerSettings::for_project("gas", _project)?;
        let Some(settings) = settings.settings else {
            return Err("missing gas settings".into());
        };
        let settings: GitHubActivitySummarizerServerSettings =
            serde_json::from_value(settings).map_err(|e| e.to_string())?;

        let mut env_vars = vec![];

        if let Some(query_extra) = settings.query_extra {
            env_vars.push(("GITHUB_GAS_QUERY_EXTRA".into(), query_extra));
        }

        if let Some(repositories) = settings.repositories {
            env_vars.push(("GITHUB_GAS_REPOSITORIES".into(), repositories.join(",")));
        }

        if let Some(from_date) = settings.from_date {
            env_vars.push(("GITHUB_GAS_FROM_DATE".into(), from_date));
        }

        env_vars.push(("GITHUB_GAS_AUTHOR".into(), settings.author));


        let downloaded = self.download()?;
        let current_dir = std::env::current_dir().map_err(|e| e.to_string())?;
        let bin_path = current_dir.join(downloaded.path);

        Ok(Command {
            command: bin_path.to_string_lossy().to_string(),
            args: vec!["stdio".to_string()],
            env: env_vars,
        })
    }
}

struct GitHubGASExtension {
    cached_binary_path: Option<String>,
}

#[derive(Clone)]
struct GitHubGASServerBinary {
    path: String,
}

impl GitHubGASExtension {
    fn download(
        &mut self,
    ) -> Result<GitHubGASServerBinary> {
        // write something to stderr
        eprintln!("Downloading binary to {BINARY_NAME}");

        let binary_path = format!("{BINARY_NAME}");

        if fs::metadata(&binary_path).map_or(false, |stat| stat.is_file()) {
            return Ok(GitHubGASServerBinary {
                path: binary_path.clone(),
            });
        }

        let release = zed::latest_github_release(
            "rubiojr/gas",
            GithubReleaseOptions {
                require_assets: true,
                pre_release: false,
            },
        )?;

        let (platform, arch) = zed::current_platform();
        let asset_name = format!(
            "{BINARY_NAME}-{os}-{arch}{extension}",
            arch = match arch {
                zed::Architecture::Aarch64 => "arm64",
                zed::Architecture::X8664 => "amd64",
                _ => "unknown",
            },
            os = match platform {
                zed::Os::Mac => "darwin",
                zed::Os::Linux => "linux",
                zed::Os::Windows => "windows",
            },
            extension = match platform {
                zed::Os::Windows => ".ext",
                _ => "",
            }
        );

        let asset = release
            .assets
            .iter()
            .find(|asset| asset.name == asset_name)
            .ok_or_else(|| format!("no asset found matching {:?}", asset_name))?;

        if !fs::metadata(&binary_path).map_or(false, |stat| stat.is_file()) {
            zed::download_file(
                &asset.download_url,
                &binary_path,
                match platform {
                    zed::Os::Mac | zed::Os::Linux => zed::DownloadedFileType::Uncompressed,
                    zed::Os::Windows => zed::DownloadedFileType::Uncompressed,
                },
            )
            .map_err(|e| format!("failed to download file: {e}"))?;

            zed::make_file_executable(&binary_path)?;

            let entries =
                fs::read_dir(".").map_err(|e| format!("failed to list working directory {e}"))?;
            for entry in entries {
                let entry = entry.map_err(|e| format!("failed to load directory entry {e}"))?;
                if entry.file_name().to_str() != Some(&binary_path) {
                    fs::remove_dir_all(&entry.path()).ok();
                }
            }
        }

        self.cached_binary_path = Some(binary_path.clone());
        Ok(GitHubGASServerBinary {
            path: binary_path,
        })
    }
}

zed::register_extension!(GitHubGASExtension);
