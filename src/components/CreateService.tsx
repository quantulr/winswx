import {
  Button,
  FormControl,
  FormErrorMessage,
  FormLabel,
  IconButton,
  Input,
  InputGroup,
  InputRightElement,
  useToast,
} from "@chakra-ui/react";
import { MdArrowBack } from "react-icons/md";
import { useNavigate } from "react-router-dom";
import { useFormik } from "formik";
import * as Yup from "yup";
import { CgSelectR } from "react-icons/cg";
import { dialog } from "@tauri-apps/api";

const ServiceSchema = Yup.object().shape({
  id: Yup.string().required("请填写服务id"),
  executable: Yup.string().required("请选择可执行文件路径"),
});

const CreateService = () => {
  const navigate = useNavigate();
  const toast = useToast();
  const formik = useFormik({
    initialValues: {
      id: "",
      executable: "",
    },
    validationSchema: ServiceSchema,
    onSubmit: (values) => {
      toast({
        position: "top",
        title: values.id,
      });
    },
  });
  return (
    <>
      <div className={"p-2"}>
        <Button
          size={"sm"}
          leftIcon={<MdArrowBack />}
          onClick={() => navigate("/")}
        >
          返回
        </Button>
      </div>
      <div className={"px-12 mt-8"}>
        <form onSubmit={formik.handleSubmit}>
          <FormControl isInvalid={!!formik.errors.id && formik.touched.id}>
            <FormLabel>服务ID</FormLabel>
            <Input
              id={"id"}
              onChange={formik.handleChange}
              value={formik.values.id}
            />
            <FormErrorMessage>{formik.errors.id}</FormErrorMessage>
          </FormControl>
          <FormControl
            isInvalid={!!formik.errors.executable && formik.touched.executable}
          >
            <FormLabel>可执行文件</FormLabel>
            <InputGroup>
              <Input
                id={"executable"}
                onChange={formik.handleChange}
                value={formik.values.executable}
              />
              <InputRightElement>
                <IconButton
                  size={"xs"}
                  aria-label={""}
                  icon={<CgSelectR />}
                  onClick={() => {
                    dialog
                      .open({
                        filters: [
                          {
                            name: "executable",
                            extensions: ["exe"],
                          },
                        ],
                      })
                      .then((file) => {
                        toast({
                          position: "top",
                          title: file,
                        });
                      });
                  }}
                />
              </InputRightElement>
            </InputGroup>
            <FormErrorMessage>{formik.errors.executable}</FormErrorMessage>
          </FormControl>
          <div className={"flex justify-center items-center mt-2"}>
            <Button type={"submit"}>新增</Button>
          </div>
        </form>
      </div>
    </>
  );
};

export default CreateService;
