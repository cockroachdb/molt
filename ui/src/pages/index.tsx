import FetchList from './FetchList';
import ConfigureTask from './ConfigureTask';
import SetupConnection from './connections/SetupConnection';
import DetailedFetch from './DetailedFetch';

export interface RouteEntry {
    path: string;
    name: string;
    element: JSX.Element;
}

export const HOME_PATH = "/";
export const FETCH_HOME_PATH = "/fetch";
export const DETAILED_FETCH_PATH = "/fetch/:fetchId";
export const CONFIGURE_TASK_PATH = "/configure-task";
export const SETUP_CONNECTION_PATH = "/setup-connection";

export const ROUTES: RouteEntry[] = [
    {
        path: HOME_PATH,
        name: "Fetch List",
        element: <FetchList />,
    },
    {
        path: DETAILED_FETCH_PATH,
        name: "Detailed Fetch Page",
        element: <DetailedFetch />,
    },
    {
        path: CONFIGURE_TASK_PATH,
        name: "Configure Task",
        element: <ConfigureTask />,
    },
    {
        path: SETUP_CONNECTION_PATH,
        name: "Setup Connection",
        element: <SetupConnection />,
    },
]