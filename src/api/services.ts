import {BaseDirectory, createDir, exists, readDir} from "@tauri-apps/api/fs";

const checkServicesDirExist = async () => {
    const isExist = await exists("service", {dir: BaseDirectory.AppData})
    if (!isExist) {
        await createDir('services', {dir: BaseDirectory.AppData, recursive: true})
    }
}

export const servicesList = async () => {
    await checkServicesDirExist()
    const entries = await readDir('services', {dir: BaseDirectory.AppData, recursive: true});
    console.log(entries)
    return entries
}