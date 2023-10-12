use std::{fs::File, io::Write};

use quick_xml::events::Event;
use quick_xml::reader::Reader;
use serde::{Deserialize, Serialize};
use serde_json::{Result, Value};
use serde_xml_rs;
use tauri::PathResolver;
fn read_xml(path: &str) {
    let reader = Reader::from_file(path);
}

#[derive(Serialize, Deserialize)]
#[serde(rename = "service")]
struct Service {
    id: String,
    executable: String,
}

#[tauri::command]
pub fn write_xml(content: &str, path: &str) {
    tauri::Builder::default().setup(|app| {
        let resource_path = app.path_resolver().app_data_dir().expect("err");
        println!("{}", &resource_path.to_str().unwrap());
        Ok(())
    });
    let service: Service = serde_json::from_str(content).unwrap();
    let result = serde_xml_rs::to_string(&service).unwrap();
    let mut file = File::create(path).unwrap();
    file.write_all(&result.as_bytes());
}
