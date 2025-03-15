struct GitHubActivitySummarizerCS;
use serde::Deserialize;
use zed::settings::ContextServerSettings;
use zed_extension_api::{self as zed, serde_json, Command, ContextServerId, Project, Result};

#[derive(Debug, Deserialize)]
struct GitHubActivitySummarizerServerSettings {
    repositories: Option<Vec<String>>,
    query_extra: Option<String>,
    from_date: Option<String>,
}

impl zed::Extension for GitHubActivitySummarizerCS {
    fn new() -> Self {
        Self
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

        let mut env_vars = vec![
        ];

        if let Some(query_extra) = settings.query_extra {
            env_vars.push(("GITHUB_GAS_QUERY_EXTRA".into(), query_extra));
        }

        if let Some(repositories) = settings.repositories {
            env_vars.push(("GITHUB_GAS_REPOSITORIES".into(), repositories.join(",")));
        }

        if let Some(from_date) = settings.from_date{
            env_vars.push(("GITHUB_GAS_FROM_DATE".into(), from_date));
        }

        Ok(Command {
            command: "github-gas-server".to_string(),
            args: vec!["stdio".to_string()],
            env: env_vars,
        })
    }
}

zed::register_extension!(GitHubActivitySummarizerCS);
