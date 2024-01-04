import {
  Badge,
  Button,
  Menu,
  MenuButton,
  MenuItem,
  MenuList,
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
import { useNavigate } from "react-router-dom";
import { LuFileEdit } from "react-icons/lu";
import { HiChevronDown } from "react-icons/hi2";

const Services = () => {
  const navigate = useNavigate();
  const toast = useToast();
  const { services /*, isLoading*/, isError, mutate } = useServicesList();
  if (isError) {
    return <div>error</div>;
  }
  return (
    <div className={"p-6"}>
      <div>
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
      <TableContainer className={"mt-2"}>
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
                  <Menu>
                    <MenuButton
                      size={"xs"}
                      as={Button}
                      rightIcon={<HiChevronDown />}
                    >
                      操作
                    </MenuButton>
                    <MenuList>
                      {service.status === "NonExistent" && (
                        <MenuItem
                          onClick={() => {
                            winswCommand("install", service.path).then(() => {
                              void mutate();
                            });
                          }}
                          icon={<MdInstallDesktop />}
                        >
                          安装
                        </MenuItem>
                      )}
                      {service.status === "Inactive (stopped)" && (
                        <MenuItem
                          onClick={() => {
                            winswCommand("uninstall", service.path).then(() => {
                              void mutate();
                            });
                          }}
                          icon={<MdDeleteOutline />}
                        >
                          卸载
                        </MenuItem>
                      )}
                      {service.status === "Inactive (stopped)" && (
                        <MenuItem
                          onClick={() => {
                            winswCommand("start", service.path).then(() => {
                              void mutate();
                            });
                          }}
                          icon={<VscDebugStart />}
                        >
                          启动
                        </MenuItem>
                      )}
                      {service.status === "Active (running)" && (
                        <MenuItem
                          onClick={() => {
                            winswCommand("stop", service.path).then(() => {
                              void mutate();
                            });
                          }}
                          icon={<HiOutlineStop />}
                        >
                          停止
                        </MenuItem>
                      )}
                      {service.status == "Active (running)" && (
                        <MenuItem
                          onClick={() => {
                            winswCommand("restart", service.path).then(() => {
                              void mutate();
                            });
                          }}
                          icon={<MdOutlineRestartAlt />}
                        >
                          重启
                        </MenuItem>
                      )}
                      {service.status != "NonExistent" && (
                        <MenuItem
                          onClick={() => {
                            winswCommand("refresh", service.path).then(() => {
                              void mutate();
                            });
                          }}
                          icon={<MdOutlineRefresh />}
                        >
                          刷新
                        </MenuItem>
                      )}
                      <MenuItem
                        onClick={() => {
                          navigate(`/edit/${service.id}`);
                        }}
                        icon={<LuFileEdit />}
                      >
                        修改
                      </MenuItem>
                      {service.status === "NonExistent" && (
                        <MenuItem
                          onClick={() => {
                            removeService(service.id).then(() => {
                              toast({
                                title: `服务已删除`,
                                status: "success",
                              });
                              void mutate();
                            });
                          }}
                          icon={<MdDeleteOutline />}
                        >
                          删除
                        </MenuItem>
                      )}
                    </MenuList>
                  </Menu>
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
