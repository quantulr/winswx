import { createBrowserRouter, Outlet /*useLocation*/ } from "react-router-dom";

import CreateService from "./components/CreateService.tsx";
import ServicesGrid from "./components/ServicesGrid.tsx";
import { OverlayScrollbarsComponent } from "overlayscrollbars-react";

/*const TransitionRoute = ({
                             path,
                             children,
                         }: {
    path: string;
    children: ReactNode;
}) => {
    const {pathname} = useLocation();
    const isActive = useMemo(() => path === pathname, [path, pathname]);
    return (
        <ScaleFade initialScale={0.8} in={isActive}>
            {children}
        </ScaleFade>
    );
};*/
/*const _Shell = ({children}: { children: ReactNode }) => {
    const ref = useRef<HTMLDivElement | null>(null);
    const [initialize, instance] = useOverlayScrollbars({});
    useEffect(() => {
        if (ref.current !== null) initialize(ref.current);
        return () => {
            instance()?.destroy();
            ref.current = null;
        };
    }, [initialize]);
    return (
        <div ref={ref} className={"h-screen"}>
            {children}
        </div>
    );
};*/

const router = createBrowserRouter([
  // {
  //   path: "",
  //   element: <ServicesGrid />,
  // },
  // {
  //   path: "/new",
  //   element: <CreateService />,
  // },
  // {
  //   path: "/edit/:serviceId",
  //   element: <CreateService />,
  // },

  {
    path: "/",
    element: (
      <OverlayScrollbarsComponent
        className={"h-screen"}
        options={{
          scrollbars: {
            autoHide: "leave",
          },
        }}
      >
        <Outlet />
      </OverlayScrollbarsComponent>
    ),
    children: [
      {
        path: "",
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
    ],
  },
]);

export default router;
