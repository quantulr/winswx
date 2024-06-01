import {
  Badge,
  Button,
  Icon,
  Menu,
  MenuButton,
  MenuItem,
  MenuList,
  Progress,
  useToast,
} from "@chakra-ui/react";
import {
  MdDeleteOutline,
  MdInstallDesktop,
  MdMiscellaneousServices,
  MdOutlineRestartAlt,
} from "react-icons/md";
import { removeService, Service, winswCommand } from "../api/services.ts";
import { LuFileEdit } from "react-icons/lu";
import { useNavigate } from "react-router-dom";
import { useState } from "react";

const ServicesGridItem = ({
  srv,
  onMutate,
}: {
  srv: Service;
  onMutate?: () => void;
}) => {
  const toast = useToast();
  const navigate = useNavigate();
  const [inProgress, setInProgress] = useState(false);
  return (
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
      {inProgress ? (
        <Progress
          isIndeterminate
          h={6}
          colorScheme={"purple"}
          className={"rounded-lg"}
        />
      ) : (
        <div className={"flex justify-around"}>
          <Button
            isDisabled={srv.status !== "Inactive (stopped)"}
            size={"xs"}
            onClick={() => {
              setInProgress(true);
              winswCommand("start", srv.path)
                .then(({ code }) => {
                  toast({
                    status: code === 0 ? "success" : "error",
                    title: code === 0 ? "启动成功" : "启动失败",
                  });
                  onMutate?.();
                })
                .finally(() => {
                  setInProgress(false);
                });
            }}
          >
            启动
          </Button>
          <Button
            isDisabled={srv.status !== "Active (running)"}
            size={"xs"}
            onClick={() => {
              setInProgress(true);
              winswCommand("stop", srv.path)
                .then(({ code }) => {
                  toast({
                    status: code === 0 ? "success" : "error",
                    title: code === 0 ? "停止服务成功" : "停止服务失败",
                  });
                  onMutate?.();
                })
                .finally(() => {
                  setInProgress(false);
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
                    setInProgress(true);
                    winswCommand("restart", srv.path)
                      .then(() => {
                        onMutate?.();
                      })
                      .finally(() => {
                        setInProgress(false);
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
                    setInProgress(true);
                    winswCommand("install", srv.path)
                      .then(({ code }) => {
                        toast({
                          status: code === 0 ? "success" : "error",
                          title: code === 0 ? "服务安装成功" : "服务安装失败",
                        });
                        onMutate?.();
                      })
                      .finally(() => {
                        setInProgress(false);
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
                    setInProgress(true);
                    winswCommand("uninstall", srv.path)
                      .then(({ code }) => {
                        toast({
                          status: code === 0 ? "success" : "error",
                          title: code === 0 ? "服务卸载成功" : "服务卸载失败",
                        });
                        onMutate?.();
                      })
                      .finally(() => {
                        setInProgress(false);
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
                      onMutate?.();
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
      )}
    </div>
  );
};

export default ServicesGridItem;
