import { createBrowserRouter } from "react-router-dom";
import Services from "./components/Services.tsx";
import CreateService from "./components/CreateService.tsx";
// import ServicesGrid from "./components/ServicesGrid.tsx";

const router = createBrowserRouter([
  {
    path: "/",
    element: <Services />,
  },
  {
    path: "/new",
    element: <CreateService />,
  },
  {
    path: "/edit/:serviceId",
    element: <CreateService />,
  },
]);

export default router;
