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
} from "@chakra-ui/react";
import {
  installService,
  useServicesList,
  winswCommand,
} from "../api/services.ts";
import { MdDeleteOutline, MdInstallDesktop } from "react-icons/md";
import { VscDebugStart } from "react-icons/vsc";
import { HiOutlineStop } from "react-icons/hi";

const Services = () => {
  const { services, isLoading, isError, mutate } = useServicesList();
  if (isError) {
    return <div>error</div>;
  }
  return (
    <div>
      <TableContainer>
        <Table>
          <Thead>
            <Tr>
              <Th>服务名称</Th>
              <Th>状态</Th>
              <Th>操作</Th>
            </Tr>
          </Thead>
          <Tbody>
            {services?.map((service) => (
              <Tr key={service.path}>
                <Td>{service.name}</Td>
                <Td>
                  <Badge>{service.status}</Badge>
                </Td>
                <Td>
                  {service.status === "NonExistent" && (
                    <Button
                      size={"xs"}
                      leftIcon={<MdInstallDesktop />}
                      onClick={() => {
                        installService(service.path).then(() => {
                          mutate();
                        });
                      }}
                    >
                      安装
                    </Button>
                  )}
                  {service.status === "Inactive (stopped)" && (
                    <Button
                      size={"xs"}
                      leftIcon={<MdDeleteOutline />}
                      onClick={() => {
                        winswCommand("uninstall", service.path).then(() => {
                          mutate();
                        });
                      }}
                    >
                      卸载
                    </Button>
                  )}
                  {service.status === "Inactive (stopped)" && (
                    <Button
                      size={"xs"}
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
                      size={"xs"}
                      leftIcon={<HiOutlineStop />}
                      onClick={() => {
                        winswCommand("stop", service.path).then(() => {
                          mutate();
                        });
                      }}
                    >
                      停止
                    </Button>
                  )}
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
