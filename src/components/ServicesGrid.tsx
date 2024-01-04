import {useNavigate} from "react-router-dom";
import {Icon, useToast} from "@chakra-ui/react";
import {useServicesList} from "../api/services.ts";
import {GoPlus} from "react-icons/go";

const ServicesGrid = () => {
    const navigate = useNavigate();
    const toast = useToast();
    const {services /*, isLoading*/, isError, mutate} = useServicesList();
    if (isError) {
        return <div>error</div>;
    }
    return (
        <div className={"p-6"}>
            <div className={"services-grid grid grid-cols-2 md:grid-cols-4 xl:grid-cols-6 gap-3"}>
                {services?.map((srv) => (
                    <div
                        className={
                            "aspect-[3/2] bg-white shadow cursor-pointer hover:shadow-lg transition-shadow rounded-lg p-2"
                        }
                    >
                        <h3 className={"font-bold text-base truncate"}>{srv.id}</h3>
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
                    <Icon as={GoPlus} boxSize={12}/>
                </div>
            </div>
        </div>
    );
};

export default ServicesGrid;
