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
import { isNil, omitBy } from "lodash-es";
import { MdArrowBack } from "react-icons/md";
import { useNavigate } from "react-router-dom";
import { useFormik } from "formik";
import * as Yup from "yup";
import { CgSelectR } from "react-icons/cg";
import { dialog } from "@tauri-apps/api";
import { ServiceDetail, writeService } from "../api/services.ts";
import { appDataDir, join } from "@tauri-apps/api/path";

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
      arguments: "",
    },
    validationSchema: ServiceSchema,
    onSubmit: async (values) => {
      const appDataPath = await appDataDir();
      const servicePath = await join(
        appDataPath,
        "services",
        values.id,
        `${values.id}.xml`,
      );
      const service = omitBy(values, (value) => isNil(value) || value === "");
      await writeService(service as unknown as ServiceDetail, servicePath);
      toast({
        title: JSON.stringify(
          omitBy(values, (value) => isNil(value) || value === ""),
        ),
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
            {formik.errors.id && formik.touched.id ? (
              <FormErrorMessage>{formik.errors.id}</FormErrorMessage>
            ) : (
              <div className={"h-[25px]"}></div>
            )}
          </FormControl>
          <FormControl
            isInvalid={!!formik.errors.executable && formik.touched.executable}
          >
            <FormLabel>可执行文件路径</FormLabel>
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
                        if (file) {
                          formik.setFieldValue("executable", file);
                        }
                      });
                  }}
                />
              </InputRightElement>
            </InputGroup>

            {formik.errors.executable && formik.touched.executable ? (
              <FormErrorMessage>{formik.errors.executable}</FormErrorMessage>
            ) : (
              <div className={"h-[25px]"}></div>
            )}
          </FormControl>
          <FormControl
            isInvalid={!!formik.errors.arguments && formik.touched.arguments}
          >
            <FormLabel>参数</FormLabel>
            <Input
              id={"arguments"}
              onChange={formik.handleChange}
              value={formik.values.arguments}
            />
            {formik.errors.arguments && formik.touched.arguments ? (
              <FormErrorMessage>{formik.errors.arguments}</FormErrorMessage>
            ) : (
              <div className={"h-[25px]"}></div>
            )}
          </FormControl>
          <div className={"flex justify-center items-center mt-2"}>
            <Button
              size={"sm"}
              px={10}
              type={"submit"}
              colorScheme={"messenger"}
            >
              新增
            </Button>
          </div>
        </form>
      </div>
    </>
  );
};

export default CreateService;
