import { useNavigate } from "react-router-dom";
import {
  // Badge,
  // Button,
  Icon,
  // Menu,
  // MenuButton,
  // MenuItem,
  // MenuList,
  Spinner,
  // useToast,
  /*useToast*/
} from "@chakra-ui/react";
import {
  // removeService,
  useServicesList,
  // winswCommand,
} from "../api/services.ts";
import { GoPlus } from "react-icons/go";
// import {
//     MdDeleteOutline,
//     MdInstallDesktop,
//     MdMiscellaneousServices,
//     MdOutlineRestartAlt,
// } from "react-icons/md";
// import {LuFileEdit} from "react-icons/lu";
import ServiceGridItem from "./ServiceGridItem.tsx";

const ServicesGrid = () => {
  const navigate = useNavigate();
  const { services, isLoading, isError, mutate } = useServicesList();
  // const toast = useToast();
  if (isError) {
    return (
      <div className={"gradient h-screen flex justify-center items-center"}>
        error
      </div>
    );
  }
  if (isLoading)
    return (
      <div className={"gradient min-h-screen flex justify-center items-center"}>
        <Spinner />
      </div>
    );
  return (
    <div className={`gradient p-6 min-h-screen`}>
      <div
        className={
          "services-grid grid grid-cols-2 md:grid-cols-4 xl:grid-cols-6 gap-3"
        }
      >
        {services?.map((srv) => (
          <ServiceGridItem
            key={srv.id}
            srv={srv}
            onMutate={() => {
              void mutate();
            }}
          />
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
