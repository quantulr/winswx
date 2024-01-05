import { useNavigate } from "react-router-dom";
import {
  Badge,
  Button,
  Icon,
  Menu,
  MenuButton,
  MenuItem,
  MenuList,
  Spinner,
  useToast /*useToast*/,
} from "@chakra-ui/react";
import {
  removeService,
  useServicesList,
  winswCommand,
} from "../api/services.ts";
import { GoPlus } from "react-icons/go";
import {
  MdDeleteOutline,
  MdInstallDesktop,
  MdMiscellaneousServices,
  MdOutlineRestartAlt,
} from "react-icons/md";
import { LuFileEdit } from "react-icons/lu";

const ServicesGrid = () => {
  const navigate = useNavigate();
  const { services, isLoading, isError, mutate } = useServicesList();
  const toast = useToast();
  if (isError) {
    return <div className={"gradient h-screen"}>error</div>;
  }
  if (isLoading)
    return (
      <div className={"gradient h-screen flex justify-center items-center"}>
        <Spinner />
      </div>
    );
  return (
    <div className={`gradient p-6 h-screen`}>
      <div
        className={
          "services-grid grid grid-cols-2 md:grid-cols-4 xl:grid-cols-6 gap-3"
        }
      >
        {services?.map((srv) => (
          <div
            key={srv.id}
            className={
              "aspect-[3/2] bg-white shadow hover:shadow-2xl transition-shadow rounded-lg p-2 flex flex-col justify-between"
            }
          >
            <div className={"flex justify-between items-center"}>
              <h3 className={"font-bold text-base truncate"}>{srv.id}</h3>
              <Icon as={MdMiscellaneousServices} />
            </div>
            <Badge
              className={"self-center !rounded select-none"}
              colorScheme={
                srv.status === "Active (running)"
                  ? "green"
                  : srv.status === "Inactive (stopped)"
                    ? "red"
                    : undefined
              }
            >
              {srv.status}
            </Badge>
            <div className={"flex justify-around"}>
              <Button
                isDisabled={srv.status !== "Inactive (stopped)"}
                size={"xs"}
                onClick={() => {
                  winswCommand("start", srv.path).then(() => {
                    void mutate();
                  });
                }}
              >
                启动
              </Button>
              <Button
                isDisabled={srv.status !== "Active (running)"}
                size={"xs"}
                onClick={() => {
                  winswCommand("stop", srv.path).then(() => {
                    void mutate();
                  });
                }}
              >
                停止
              </Button>
              <Menu>
                <MenuButton as={Button} size={"xs"}>
                  更多
                </MenuButton>
                <MenuList>
                  {srv.status === "Active (running)" && (
                    <MenuItem
                      onClick={() => {
                        winswCommand("restart", srv.path).then(() => {
                          void mutate();
                        });
                      }}
                      icon={<MdOutlineRestartAlt />}
                    >
                      重启
                    </MenuItem>
                  )}
                  {srv.status === "NonExistent" && (
                    <MenuItem
                      onClick={() => {
                        winswCommand("install", srv.path).then(() => {
                          void mutate();
                        });
                      }}
                      icon={<MdInstallDesktop />}
                    >
                      安装
                    </MenuItem>
                  )}
                  {srv.status === "Inactive (stopped)" && (
                    <MenuItem
                      icon={<MdDeleteOutline />}
                      onClick={() => {
                        winswCommand("uninstall", srv.path).then(() => {
                          void mutate();
                        });
                      }}
                    >
                      卸载
                    </MenuItem>
                  )}
                  <MenuItem
                    onClick={() => {
                      navigate(`/edit/${srv.id}`);
                    }}
                    icon={<LuFileEdit />}
                  >
                    修改
                  </MenuItem>{" "}
                  {srv.status === "NonExistent" && (
                    <MenuItem
                      onClick={() => {
                        removeService(srv.id).then(() => {
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
            </div>
          </div>
        ))}
        <div
          onClick={() => {
            navigate("/new");
          }}
          className={
            "aspect-[3/2] cursor-pointer flex justify-center items-center bg-white shadow hover:shadow-lg transition-shadow rounded-lg p-2"
          }
        >
          <Icon as={GoPlus} boxSize={12} />
        </div>
      </div>
    </div>
  );
};

export default ServicesGrid;
