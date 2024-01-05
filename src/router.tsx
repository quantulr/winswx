import { createBrowserRouter /*useLocation*/ } from "react-router-dom";

import CreateService from "./components/CreateService.tsx";

// import {ReactNode, useMemo} from "react";
// import {ScaleFade} from "@chakra-ui/react";
import ServicesGrid from "./components/ServicesGrid.tsx";

// const TransitionRoute = ({
//                              path,
//                              children,
//                          }: {
//     path: string;
//     children: ReactNode;
// }) => {
//     const {pathname} = useLocation();
//     const isActive = useMemo(() => path === pathname, [path, pathname]);
//     return (
//         <ScaleFade initialScale={0.8} in={isActive}>
//             {children}
//         </ScaleFade>
//     );
// };

const router = createBrowserRouter([
  {
    path: "/",
    element: <ServicesGrid />,
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
