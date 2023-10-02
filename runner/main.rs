use std::process::{exit, Command};

fn main() {
    match Command::new("bash")
        .arg(env!("SCRIPT_PATH"))
        .args(std::env::args().skip(1))
        .current_dir(env!("DIR"))
        .status()
    {
        Ok(status) => {
            let Some(code) = status.code() else {
                eprintln!("No exit code: {}", status);
                exit(0)
            };
            exit(code)
        }
        Err(e) => {
            eprintln!("error running command: {}", e);
            exit(1)
        }
    }
}
