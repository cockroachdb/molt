import { grey } from '@mui/material/colors';
import { useNavigate } from "react-router-dom";
import { useState } from 'react';
import {
    Typography,
    Box,
    MenuItem,
    Button,
    SelectChangeEvent,
} from '@mui/material';

import { MuiMarkdown, getOverrides } from 'mui-markdown';
import { AddConnection } from './CreateConnection';
import { CONFIGURE_TASK_PATH } from '..';
import { SelectGroup } from '../../components';

export interface Connection {
    id: string;
    name: string;
    dialect?: string;
    host?: string;
    port?: number;
    username?: string;
    password?: string; // this won't ever be stored (should always be empty); should be populated before sending request out.
    databaseName?: string;
    sslMode?: string;
}

const mockConnections: Connection[] = [
    {
        id: "1",
        name: "rluu-pg-to-crdb"
    },
    {
        id: "2",
        name: "jyang-pg-to-crdb"
    }]

const createConnectionMap = (connections: Connection[]): Map<string, Connection> => {
    const connMap = new Map<string, Connection>();

    connections.map(item => {
        return connMap.set(item.id, item);
    })

    return connMap;
}


export default function SetupConnection() {
    const navigate = useNavigate();
    const [showAddConnection, setShowAddConnection] = useState(false);
    const [sourceConnection, setSourceConnection] = useState<Connection | undefined>(mockConnections[0]);
    const [targetConnection, setTargetConnection] = useState<Connection | undefined>(mockConnections[0]);

    // TODO: use useMemo after we start fetching from server for this page.
    const connMap = createConnectionMap(mockConnections);

    return (
        <Box sx={
            {
                display: "flex",
                flexDirection: "row",
                justifyContent: "center",
                height: "100%"
            }
        }>
            <Box sx={{
                display: "flex",
                flexDirection: "column",
                alignItems: "stretch",
                flex: 1,
                py: 4,
                px: 10,
                maxWidth: "50%"
            }}>
                <Typography variant="h4" sx={{
                    mb: 5,
                }}>Setup Connections</Typography>
                <Box sx={{}}>
                    <form>
                        <Box sx={{
                            display: "flex",
                            flexDirection: "column",
                            gap: 4,
                        }}>
                            <SelectGroup
                                required
                                label="Source Connection"
                                id="source-connection"
                                value={sourceConnection?.id}
                                onChange={(event: SelectChangeEvent) => {
                                    setSourceConnection(connMap.get(event.target.value))
                                }}
                            >
                                {mockConnections.map(item => {
                                    return <MenuItem key={item.name} value={item.id}>{item.name}</MenuItem>
                                })}
                            </SelectGroup>
                            <SelectGroup
                                required
                                label="Target Connection"
                                id="target-connection"
                                value={targetConnection?.id}
                                onChange={(event: SelectChangeEvent) => {
                                    setTargetConnection(connMap.get(event.target.value))
                                }}
                                notched={true}
                            >
                                {mockConnections.map(item => {
                                    return <MenuItem key={item.name} value={item.id}>{item.name}</MenuItem>
                                })}
                            </SelectGroup>
                        </Box>
                        <Box sx={{
                            display: "flex",
                            flexDirection: "row",
                            justifyContent: "flex-end",
                            my: 2,
                            gap: 1,
                        }}>
                            <Button onClick={(e) => setShowAddConnection(true)} variant="outlined">Add Connection</Button>
                            <Button onClick={() => navigate(CONFIGURE_TASK_PATH)} variant="contained">Next</Button>
                        </Box>
                    </form>
                </Box >
                {
                    showAddConnection && <AddConnection hideAddConnection={() => setShowAddConnection(false)} />
                }
            </Box>
            <Box sx={{
                display: "flex",
                flexDirection: "column",
                flex: 1,
                backgroundColor: grey[50],
                py: 4,
                px: 6,
                gap: 4,
            }}>
                <Typography variant="h4">Setup Guide</Typography>
                <MuiMarkdown overrides={{
                    ...getOverrides(), // This will keep the other default overrides.
                    code: {
                        props: {
                            style: { fontSize: "0.8rem", backgroundColor: grey[200] },
                        } as React.HTMLProps<HTMLParagraphElement>,
                    },
                    p: {
                        props: {
                            style: { fontSize: "0.95rem" },
                        } as React.HTMLProps<HTMLParagraphElement>,
                    },
                    strong: {
                        props: {
                            style: { fontSize: "0.95rem" },
                        } as React.HTMLProps<HTMLParagraphElement>,
                    },
                }}>
                    {`##### Use existing connections
Select an existing connection for both the source (\`Source Connection\`) and target connection (\`Target Connection\`). 
Just a note that you must select different connections.

<br/>

##### Adding new connections
In order to setup new connections, you must first have a running database that is available over the internet, or locally (if you are running locally).
<br/><br/>Here are the details we will need from you:
- **Credential Name**: unique name for the credential
- **Dialect**: dialect for the database (MySQL, Cockroach, PostgreSQL)
- **Host**: host URI or IP address of the server running the database (i.e. \`http://fetch.com\`, \`https://127.0.0.1/2333\`)
- **Port**: port number for the database process
- **Username**: name of the SQL user
- **Password**: password for the SQL user
- **Database Name**: name of the database to access
- **SSL Mode**: SSL setting for the database (dictates if a cert must be supplied or not)

<br/>
You can obtain the details from the connection string. For example:
\`\`\`
postgres://postgres:postgres@localhost:5432/molt?sslmode=disable
\`\`\`
`}
                </MuiMarkdown>
            </Box>
        </Box >
    )
}