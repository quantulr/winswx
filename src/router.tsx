import { createBrowserRouter } from "react-router-dom";
import Services from "./components/Services.tsx";
import CreateService from "./components/CreateService.tsx";

const router = createBrowserRouter([
  {
    path: "/",
    element: <Services />,
  },
  {
    path: "/new",
    element: <CreateService />,
  },
]);

export default router;