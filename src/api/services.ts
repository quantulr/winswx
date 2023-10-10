import { BaseDirectory, createDir, exists, readDir } from "@tauri-apps/api/fs";
import useSWR from "swr";
import { join } from "@tauri-apps/api/path";
import { Command } from "@tauri-apps/api/shell";

const checkServicesDirExist = async () => {
  const isExist = await exists("service", { dir: BaseDirectory.AppData });
  if (!isExist) {
    await createDir("services", {
      dir: BaseDirectory.AppData,
      recursive: true,
    });
  }
};

interface Service {
  name?: string;
  path: string;
  status: string;
}

const serviceStatus = async (path: string) => {
  const command = Command.sidecar("bin/winsw", ["status", path]);
  const child = await command.execute();
  if (child.code !== 0) {
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
  const { data, isLoading, error } = useSWR("/services", fetcher);
  console.log(data);
  return {
    services: data,
    isLoading,
    isError: error,
  };
};
