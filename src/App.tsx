import "./App.css";
import { ChakraProvider } from "@chakra-ui/react";
import { RouterProvider } from "react-router-dom";
import router from "./router.tsx";

function App() {
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
