import {
    Typography,
    Box,
    MenuItem,
    Button,
    IconButton,
    SelectChangeEvent,
} from '@mui/material';
import CloseIcon from '@material-ui/icons/Close';
import { FormEvent, useState } from 'react';
import { InputGroup, SelectGroup } from '../../components';

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

interface AddConnectionProps {
    hideAddConnection: () => void;
}

const dialectToPortMapping: Record<string, number> = {
    "CockroachDB": 26257,
    "PostgreSQL": 5432,
    "MySQL": 3306,

}
const dialects = ["CockroachDB", "PostgreSQL", "MySQL"]
// TODO: add SSL mode and the uploader for SSL mode file.
const SSLModes = ["Disable"]


export const AddConnection = (props: AddConnectionProps) => {
    const [credName, setCredName] = useState("");
    const [dialect, setDialect] = useState(dialects[0]);
    const [host, setHost] = useState("");
    const [port, setPort] = useState(dialectToPortMapping[dialects[0]]);
    const [username, setUsername] = useState("");
    const [password, setPassword] = useState("");
    const [dbName, setDbName] = useState("");
    const [SSLMode, setSSLMode] = useState(SSLModes[0]);

    const handleSubmit = (e: FormEvent) => {
        e.preventDefault();
        alert(credName)
    }

    return <Box>
        <Box sx={{
            display: "flex",
            flexDirection: "row",
            gap: 1,
            mb: 4,
        }}>
            <Typography variant="h4" alignSelf={"center"}>Add a new connection</Typography>
            <IconButton onClick={props.hideAddConnection}>
                <CloseIcon />
            </IconButton>
        </Box>
        <Box>
            <form onSubmit={handleSubmit}>
                <Box sx={{
                    display: "flex",
                    flexDirection: "column",
                    gap: 4,
                }}>
                    <InputGroup
                        label={"Credential Name"}
                        required={true}
                        autoFocus={true}
                        id="credName"
                        value={credName}
                        validation={(value) => {
                            if (value.length === 0) return "Field cannot be empty."

                            return ""
                        }}
                        onChange={(e) => setCredName(e.target.value)} />
                    <SelectGroup
                        required
                        label="Dialect"
                        id="dialect"
                        value={dialect}
                        onChange={(event: SelectChangeEvent) => {
                            setDialect(event.target.value);
                            setPort(dialectToPortMapping[event.target.value])
                        }}
                    >
                        {dialects.map(item => {
                            return <MenuItem key={item} value={item}>{item}</MenuItem>
                        })}
                    </SelectGroup>
                    <Box sx={{
                        display: "flex",
                        flexDirection: "row",
                        gap: 2,
                    }}>
                        <InputGroup
                            required
                            id="host"
                            label="Host"
                            value={host}
                            validation={(value) => {
                                if (value.length === 0) return "Field cannot be empty."

                                return ""
                            }}
                            onChange={(e) => setHost(e.target.value)} />
                        <InputGroup
                            required
                            label="Port"
                            type='number'
                            id="port"
                            value={port}
                            validation={(value) => {
                                if (value.length === 0) return "Field cannot be empty."

                                return ""
                            }}
                            onChange={(e) => setPort(parseInt(e.target.value))} />
                    </Box>
                    <Box sx={{
                        display: "flex",
                        flexDirection: "row",
                        gap: 2,
                    }}>
                        <InputGroup
                            required
                            label="Username"
                            id="username"
                            value={username}
                            validation={(value) => {
                                if (value.length === 0) return "Field cannot be empty."

                                return ""
                            }}
                            onChange={(e) => setUsername(e.target.value)} />
                        <InputGroup
                            required
                            label="Password"
                            type='password'
                            id="password"
                            value={password}
                            validation={(value) => {
                                if (value.length === 0) return "Field cannot be empty."

                                return ""
                            }}
                            onChange={(e) => setPassword(e.target.value)} />
                    </Box>
                    <Box sx={{
                        display: "flex",
                        flexDirection: "row",
                        gap: 2,
                    }}>
                        <InputGroup
                            required
                            label="Database Name"
                            id="dbName"
                            value={dbName}
                            validation={(value) => {
                                if (value.length === 0) return "Field cannot be empty."

                                return ""
                            }}
                            onChange={(e) => setDbName(e.target.value)} />
                        <SelectGroup
                            required
                            label="SSL Mode"
                            id="sslmode"
                            size="small"
                            value={SSLMode}
                            onChange={(event: SelectChangeEvent) => {
                                setSSLMode(event.target.value);
                            }}
                        >
                            {SSLModes.map(item => {
                                return <MenuItem key={item} value={item}>{item}</MenuItem>
                            })}
                        </SelectGroup>
                    </Box>
                    <Button type="submit" variant="contained">Add</Button>
                </Box>
            </form>
        </Box>
    </Box>
}