import {
  BaseDirectory,
  copyFile,
  createDir,
  exists,
  readDir,
  readTextFile,
  removeDir,
  writeTextFile,
} from "@tauri-apps/api/fs";
import useSWR from "swr";
import { dirname, join, resolveResource } from "@tauri-apps/api/path";
import { Command } from "@tauri-apps/api/shell";
import { XMLBuilder, XMLParser } from "fast-xml-parser";

/**
 * 检查服务文件父级目录是否存在，不存在则创建
 */
const checkServicesDirExist = async () => {
  const isExist = await exists("service", { dir: BaseDirectory.AppData });
  if (!isExist) {
    await createDir("services", {
      dir: BaseDirectory.AppData,
      recursive: true,
    });
  }
};

/**
 *
 */
export const copyBinary = async () => {
  const binName = "winsw-x86_64-pc-windows-msvc.exe";
  // const isAppDataDirExist = await exists()
  await checkServicesDirExist();
  const resource = await resolveResource(`bin/${binName}`);
  const exist = await exists("winsw.exe", { dir: BaseDirectory.AppData });
  if (!exist) {
    await copyFile(resource, "winsw.exe", { dir: BaseDirectory.AppData });
  }
};

export interface Service {
  id: string;
  path: string;
  status: string;
}

/**
 * 获取服务状态
 * @param path
 */
const serviceStatus = async (path: string) => {
  const command = new Command("winsw", ["status", path]);
  command
    .execute()
    .then((re) => {
      console.log(re);
    })
    .catch((e) => {
      console.log(e);
    });
  const child = await command.execute();
  console.log(233);
  if (child.code === -1) {
    throw new Error(child.stdout);
  }
  return child.stdout.replace(/[\x00-\x1F\x7F-\x9F]+/g, "");
};

/**
 * 服务列表fetcher
 */
const fetcher = async (): Promise<Service[]> => {
  await checkServicesDirExist();
  const servicesDir = await readDir("services", {
    dir: BaseDirectory.AppData,
    recursive: true,
  });
  const services: Service[] = [];
  for (const fileEntry of servicesDir) {
    const id = fileEntry.name;
    if (!id) {
      throw new Error("error");
    }
    const path = await join(fileEntry.path, `${id}.xml`);
    console.log(path);
    const status = await serviceStatus(path);
    services.push({
      id,
      path,
      status,
    });
  }

  return services;
};

/**
 * 获取服务列表 swr
 */
export const useServicesList = () => {
  const { data, isLoading, error, mutate } = useSWR("/services", fetcher);

  return {
    services: data,
    isLoading,
    isError: error,
    mutate,
  };
};

/**
 * 安装service
 * @param path
 */
export const installService = async (path: string) => {
  const command = new Command("winsw", ["install", path]);
  const child = await command.execute();
  return child.stdout;
};

/**
 * 执行winsw命令
 * @param command
 * @param path
 */
export const winswCommand = async (command: string, path: string) => {
  const cmd = new Command("winsw", [command, path]);
  const child = await cmd.execute();

  return { stdout: child.stdout, code: child.code };
};

export interface ServiceDetail {
  id: string;
  executable: string;
  arguments?: string;
}

/**
 * 写入服务文件
 * @param content
 * @param path
 */
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
  return removeDir(`services/${serviceId}`, {
    dir: BaseDirectory.AppData,
    recursive: true,
  });
};

/**
 * 获取服务详情
 * @param serviceId 服务id
 */
export const useServiceDetail = (serviceId?: string) => {
  const fetcher = async (serviceId: string) => {
    const xmlContent = await readTextFile(
      `services/${serviceId}/${serviceId}.xml`,
      { dir: BaseDirectory.AppData },
    );
    const parser = new XMLParser();
    return parser.parse(xmlContent).service;
  };
  const { data, error, isLoading } = useSWR<ServiceDetail>(serviceId, fetcher);
  return {
    service: data,
    isError: error,
    isLoading,
  };
};

/**
 * 更新服务
 * @param service
 */
export const updateService = (service: ServiceDetail) => {
  const servicePath = `services/${service.id}/${service.id}.xml`;
  const isExists = exists(servicePath, { dir: BaseDirectory.AppData });
  if (!isExists) {
    throw new Error("服务不存在");
  }
  const xmlBuilder = new XMLBuilder({
    format: true,
  });
  const xmlStr = xmlBuilder.build({
    service,
  });
  return writeTextFile(servicePath, xmlStr, { dir: BaseDirectory.AppData });
};
