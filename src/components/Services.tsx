import {
  Button,
  Table,
  TableContainer,
  Tbody,
  Td,
  Th,
  Thead,
  Tr,
} from "@chakra-ui/react";
import { useServicesList } from "../api/services.ts";
import { MdInstallDesktop } from "react-icons/md";

const Services = () => {
  const { services } = useServicesList();
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
                <Td>{service.status}</Td>
                <Td>
                  <Button size={"xs"} leftIcon={<MdInstallDesktop />}>
                    安装
                  </Button>
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
