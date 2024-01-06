import "./App.css";
import { ChakraProvider, Spinner } from "@chakra-ui/react";
import { RouterProvider } from "react-router-dom";
import router from "./router.tsx";
import { useEffect, useState } from "react";
import { copyBinary } from "./api/services.ts";

import "overlayscrollbars/overlayscrollbars.css";

function App() {
  const [loading, setLoading] = useState(true);
  // useEffect(() => {
  //     const handleRightClick = (event: MouseEvent) => {
  //         event.preventDefault();
  //     };
  //     window.addEventListener("contextmenu", handleRightClick);
  //     return () => {
  //         window.removeEventListener("contextmenu", handleRightClick);
  //     };
  // }, []);

  useEffect(() => {
    copyBinary().then(() => {
      setLoading(false);
    });
  }, []);

  if (loading)
    return (
      <div className={"w-screen h-screen flex justify-center items-center"}>
        <Spinner />
      </div>
    );
  return (
    <ChakraProvider
      toastOptions={{
        defaultOptions: {
          position: "top",
        },
      }}
    >
      <RouterProvider router={router} />
    </ChakraProvider>
  );
}

export default App;
