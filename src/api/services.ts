import {BaseDirectory, createDir, exists, readDir, removeDir, writeTextFile,} from "@tauri-apps/api/fs";
import useSWR from "swr";
import {dirname, join} from "@tauri-apps/api/path";
import {Command} from "@tauri-apps/api/shell";
import {XMLBuilder} from "fast-xml-parser";

const checkServicesDirExist = async () => {
    const isExist = await exists("service", {dir: BaseDirectory.AppData});
    if (!isExist) {
        await createDir("services", {
            dir: BaseDirectory.AppData,
            recursive: true,
        });
    }
};

interface Service {
    name: string;
    path: string;
    status: string;
}

const serviceStatus = async (path: string) => {
    const command = Command.sidecar("bin/winsw", ["status", path]);
    const child = await command.execute();

    if (child.code === -1) {
        throw new Error(child.stdout);
    }
    return child.stdout.replace(/[\x00-\x1F\x7F-\x9F]+/g, "");
};

const fetcher = async (): Promise<Service[]> => {
    await checkServicesDirExist();
    const servicesDir = await readDir("services", {
        dir: BaseDirectory.AppData,
        recursive: true,
    });
    const services: Service[] = [];
    for (const fileEntry of servicesDir) {
        const name = fileEntry.name;
        if (!name) {
            throw new Error("error");
        }
        const path = await join(fileEntry.path, `${name}.xml`);
        const status = await serviceStatus(path);
        services.push({
            name,
            path,
            status,
        });
    }
    return services;
};

export const useServicesList = () => {
    const {data, isLoading, error, mutate} = useSWR("/services", fetcher);

    return {
        services: data,
        isLoading,
        isError: error,
        mutate,
    };
};

export const installService = async (path: string) => {
    const command = Command.sidecar("bin/winsw", ["install", path]);
    const child = await command.execute();
    return child.stdout;
};

export const winswCommand = async (command: string, path: string) => {
    const cmd = Command.sidecar("bin/winsw", [command, path]);
    const child = await cmd.execute();
    return child.stdout;
};

export interface ServiceDetail {
    id: string;
    executable: string;
    arguments?: string;
}

export const writeService = async (content: ServiceDetail, path: string) => {
    const xmlBuilder = new XMLBuilder({
        format: true,
    });
    const xmlStr = xmlBuilder.build({
        service: content,
    });

    const parentDir = await dirname(path);
    const isExists = await exists(parentDir);
    if (!isExists) {
        await createDir(await join(parentDir), {
            recursive: true,
        });
    }
    await writeTextFile(path, xmlStr);
};

/**
 * 删除服务
 * @param serviceId 服务id
 */
export const removeService = async (serviceId: string) => {
    return removeDir(`services/${serviceId}`, {dir: BaseDirectory.AppData, recursive: true})
}