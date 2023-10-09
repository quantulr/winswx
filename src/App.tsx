import "./App.css";
import {Command} from "@tauri-apps/api/shell";
import {Button, ChakraProvider} from "@chakra-ui/react";
import {servicesList} from "./api/services.ts";

const winswVer = async () => {
    const command = Command.sidecar("bin/winsw", ["install", "C:\\Users\\quant\\.config\\winsw\\caddy\\caddy.xml"]);

    command.on("close", (data) => {
        console.log(
            `command finished with code ${data.code} and signal ${data.signal}`,
        );
    });
    command.on("error", (error) => console.error(`command error: "${error}"`));

    command.stdout.on("data", (line) => {
        console.log(line);
    });
    command.stderr.on("data", (line) => {
        console.log(line);
    });
    await command.spawn();
};
const getAppDir = async () => {
    await servicesList()
}

function App() {
    return (
        <ChakraProvider>
            <Button onClick={() => {
                void winswVer()
            }}>安装</Button>
            <Button onClick={() => {
                void getAppDir()
            }}>dir</Button>
        </ChakraProvider>
    );
}

export default App;
