import "./App.css";
import { ChakraProvider } from "@chakra-ui/react";
import Services from "./components/Services.tsx";
// import {Command} from "@tauri-apps/api/shell";

// const winswVer = async () => {
//     const command = Command.sidecar("bin/winsw", ["status", "C:\\Users\\1quant\\.config\\winsw\\caddy\\caddy.xml"]);
//
//     command.on("close", (data) => {
//         console.log(
//             `command finished with code ${data.code} and signal ${data.signal}`,
//         );
//     });
//     command.on("error", (error) => console.error(`command error: "${error}"`));
//
//     command.stdout.on("data", (line) => {
//         console.log(line);
//     });
//     command.stderr.on("data", (line) => {
//         console.log(line);
//     });
//     const child = await command.execute();
//     console.log(child.stdout)
// };

function App() {
  return (
    <ChakraProvider>
      {/*<Button onClick={() => {*/}
      {/*    winswVer()*/}
      {/*}}></Button>*/}
      <Services />
    </ChakraProvider>
  );
}

export default App;
