import {
  Badge,
  Button,
  Table,
  TableContainer,
  Tbody,
  Td,
  Th,
  Thead,
  Tr,
  useToast,
} from "@chakra-ui/react";
import {
  installService,
  removeService,
  useServicesList,
  winswCommand,
} from "../api/services.ts";
import {
  MdAddCircleOutline,
  MdDeleteOutline,
  MdInstallDesktop,
  MdOutlineRefresh,
  MdOutlineRestartAlt,
} from "react-icons/md";
import { VscDebugStart } from "react-icons/vsc";
import { HiOutlineStop } from "react-icons/hi";
import styles from "./Services.module.scss";
import { useNavigate } from "react-router-dom";
import { LuFileEdit } from "react-icons/lu";

const Services = () => {
  const navigate = useNavigate();
  const toast = useToast();
  const { services /*, isLoading*/, isError, mutate } = useServicesList();
  if (isError) {
    return <div>error</div>;
  }
  return (
    <div>
      <div className={"p-2"}>
        <Button
          size={"sm"}
          leftIcon={<MdAddCircleOutline />}
          onClick={() => {
            navigate("/new");
          }}
        >
          新增服务
        </Button>
      </div>
      <TableContainer>
        <Table>
          <Thead>
            <Tr>
              <Th>服务ID</Th>
              <Th>状态</Th>
              <Th>操作</Th>
            </Tr>
          </Thead>
          <Tbody>
            {services?.map((service) => (
              <Tr key={service.path}>
                <Td>{service.id}</Td>
                <Td>
                  <Badge
                    colorScheme={
                      service.status === "Active (running)"
                        ? "green"
                        : service.status === "Inactive (stopped)"
                        ? "red"
                        : undefined
                    }
                  >
                    {service.status}
                  </Badge>
                </Td>
                <Td>
                  <div className={styles.operationButtons}>
                    {service.status === "NonExistent" && (
                      <Button
                        className={styles.operationButton}
                        size={"xs"}
                        colorScheme={"whatsapp"}
                        leftIcon={<MdInstallDesktop />}
                        onClick={() => {
                          installService(service.path).then(() => {
                            void mutate();
                          });
                        }}
                      >
                        安装
                      </Button>
                    )}
                    {service.status === "Inactive (stopped)" && (
                      <Button
                        className={styles.operationButton}
                        size={"xs"}
                        colorScheme={"facebook"}
                        leftIcon={<MdDeleteOutline />}
                        onClick={() => {
                          winswCommand("uninstall", service.path).then(() => {
                            void mutate();
                          });
                        }}
                      >
                        卸载
                      </Button>
                    )}
                    {service.status === "Inactive (stopped)" && (
                      <Button
                        className={styles.operationButton}
                        size={"xs"}
                        colorScheme={"green"}
                        leftIcon={<VscDebugStart />}
                        onClick={() => {
                          winswCommand("start", service.path).then(() => {
                            mutate();
                          });
                        }}
                      >
                        启动
                      </Button>
                    )}
                    {service.status === "Active (running)" && (
                      <Button
                        className={styles.operationButton}
                        size={"xs"}
                        leftIcon={<HiOutlineStop />}
                        onClick={() => {
                          winswCommand("stop", service.path).then(() => {
                            void mutate();
                          });
                        }}
                      >
                        停止
                      </Button>
                    )}
                    {service.status == "Active (running)" && (
                      <Button
                        className={styles.operationButton}
                        size={"xs"}
                        leftIcon={<MdOutlineRestartAlt />}
                        onClick={() => {
                          winswCommand("restart", service.path).then(() => {
                            void mutate();
                          });
                        }}
                      >
                        重启
                      </Button>
                    )}
                    {service.status != "NonExistent" && (
                      <Button
                        className={styles.operationButton}
                        size={"xs"}
                        leftIcon={<MdOutlineRefresh />}
                        onClick={() => {
                          winswCommand("refresh", service.path).then(() => {
                            void mutate();
                          });
                        }}
                      >
                        刷新
                      </Button>
                    )}

                    <Button
                      className={styles.operationButton}
                      size={"xs"}
                      colorScheme={"orange"}
                      leftIcon={<LuFileEdit />}
                      onClick={() => {
                        navigate(`/edit/${service.id}`);
                      }}
                    >
                      修改
                    </Button>

                    {service.status === "NonExistent" && (
                      <Button
                        className={styles.operationButton}
                        size={"xs"}
                        colorScheme={"red"}
                        leftIcon={<MdDeleteOutline />}
                        onClick={() => {
                          removeService(service.id).then(() => {
                            toast({
                              title: `服务已删除`,
                              status: "success",
                            });
                            void mutate();
                          });
                        }}
                      >
                        删除
                      </Button>
                    )}
                  </div>
                </Td>
              </Tr>
            ))}
          </Tbody>
        </Table>
      </TableContainer>
    </div>
  );
};

export default Services;
