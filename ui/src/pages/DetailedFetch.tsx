import { useState, useEffect, FormEvent } from 'react';
import { useParams } from 'react-router-dom';
import {
    Typography,
    Box,
    Button,
    Paper,
    Chip,
} from '@mui/material';
import { Search, ChevronLeft } from '@material-ui/icons';
import { useNavigate } from "react-router-dom";
import SimpleTable, { TableColumnProps } from '../components/tables/Table';
import { neutral } from '../styles/colors';
import { fontWeights } from '../styles/fonts';
import { DEFAULT_SPACING } from '../styles/theme';
import { InputGroup } from '../components';
import { getSpecificFetchTask } from '../api';

export type LogLevel = "info" | "warning" | "danger";


export interface FetchStats {
    percentComplete?: {
        description: string,
        data: number
    },
    numTables?: {
        description: string,
        data: number
    },
    numRows?: {
        description: string,
        data: number
    },
    numErrors?: {
        description: string,
        data: number
    },
}

export interface FetchLog {
    key: string;
    id: string;
    level: LogLevel;
    createdAt: string;
    message: string;
}

const mockColumns: TableColumnProps<FetchLog>[] = [
    {
        id: "createdAt",
        title: "Timestamp",
        cellStyle: { width: "15%" },
        render: (record, _) => {
            return record.createdAt
        }
    },
    {
        id: "level",
        title: "Level",
        cellStyle: { width: "15%" },
        render: (record, _) => {
            let levelLabel = record.level.toUpperCase();
            if (levelLabel === "DANGER") {
                levelLabel = "ERROR"
            }

            return <Chip size="small" label={levelLabel} variant={record.level} />
        }
    },
    {
        id: "message",
        title: "Message",
        cellStyle: { width: "70%" },
        render: (record, _) => {
            return record.message
        }
    },
];
const mockData: FetchLog[] = [
    {
        key: 'log123',
        id: '1a2b3c',
        level: 'info',
        createdAt: '2023-12-27T08:45:30',
        message: 'User 123 logged in successfully.',
    },
    {
        key: 'log456',
        id: '4d5e6f',
        level: 'danger',
        createdAt: '2023-12-27T09:15:20',
        message: 'Error: Invalid input received from client.',
    },
    {
        key: 'log789',
        id: '7g8h9i',
        level: 'info',
        createdAt: '2023-12-27T10:00:45',
        message: 'Database connection established.',
    },
    {
        key: 'logabc',
        id: 'a1b2c3',
        level: 'warning',
        createdAt: '2023-12-27T11:30:10',
        message: 'Warning: Disk space is running low.',
    },
    {
        key: 'logdef',
        id: 'd4e5f6',
        level: 'info',
        createdAt: '2023-12-27T12:45:55',
        message: 'User 456 logged out.',
    },
];

const getLevelFromString = (input: string): LogLevel => {
    switch (input) {
        case "info":
            return "info"
        case "warning":
            return "warning"
        case "error":
            return "danger"
    }

    return "info"
}

export default function DetailedFetch() {
    const { fetchId } = useParams();
    const [searchTerm, setSearchTerm] = useState("");
    const [initialLogs, setInitialLogs] = useState<FetchLog[]>([]);
    const [logs, setLogs] = useState<FetchLog[]>([]);
    const [stats, setStats] = useState<FetchStats>({});
    const navigate = useNavigate();

    // TODO: refactor this as a helper later on.
    useEffect(() => {
        const fetchData = async () => {
            console.log("fetching")
            try {
                const fid = Number(fetchId);
                const data = await getSpecificFetchTask(fid)

                const resLogs: FetchLog[] = data.logs.map(item => {
                    const createdAtTs = new Date(item.timestamp * 1000);

                    return {
                        key: data.id.toString(),
                        id: data.id.toString(),
                        level: getLevelFromString(item.level),
                        createdAt: createdAtTs.toISOString(),
                        message: item.message,
                    }
                })
                setLogs(resLogs);
                setInitialLogs(resLogs);

                const resStats: FetchStats = {
                    percentComplete: {
                        description: "% Complete",
                        data: Number(data.stats?.percent_complete)
                    },
                    numTables: {
                        description: "Number of Tables",
                        data: Number(data.stats?.num_tables)
                    },
                    numRows: {
                        description: "Number of Rows",
                        data: Number(data.stats?.num_rows)
                    },
                    numErrors: {
                        description: "Number of Errors",
                        data: Number(data.stats?.num_errors)
                    },
                }
                setStats(resStats);
            } catch (e) {
                console.error(e);
            }
        }
        fetchData()
    }, [fetchId])

    const handleSubmit = (e: FormEvent) => {
        e.preventDefault();

        if (searchTerm === "") {
            setLogs(initialLogs);
            return;
        }

        const filteredData = logs.filter(item => item.message.includes(searchTerm));
        setLogs(filteredData);
    }

    return (
        <Box sx={{
            display: "flex",
            flexDirection: "column",
            gap: 2,
            py: 4,
            px: 2
        }}>
            <Box sx={{
                display: "flex",
                flexDirection: "column",
                gap: 2,
            }}>
                <Button onClick={() => navigate(-1)} sx={{ width: DEFAULT_SPACING }} variant="icon" >
                    <ChevronLeft />
                </Button>
                <Typography sx={{ mb: 1 }} variant='h4'>Fetch Run #{fetchId}</Typography>
                <Paper sx={{
                    display: "flex",
                    flexDirection: "row",
                    justifyContent: "flex-start",
                    p: 2,
                }}>
                    {Object.keys(stats).map(key => {
                        const desc = stats[key as keyof typeof stats];
                        return <Box key={key} sx={{ borderRight: `1px solid ${neutral[400]}`, px: 2 }}>
                            <Typography color="primary" fontWeight={fontWeights["heaviest"]} variant='body1'>
                                {desc?.data}
                            </Typography>
                            <Typography fontWeight={fontWeights["light"]} variant='body2'>
                                {desc?.description}
                            </Typography>
                        </Box>
                    })}
                </Paper>
                <Paper sx={{
                    p: 2
                }}>
                    <Box sx={{
                        display: "flex",
                        flexDirection: "row",
                        gap: 1,
                    }}>
                        <Box sx={{
                            display: "flex",
                            flexDirection: "row",
                            alignItems: "center",
                            gap: 1
                        }} component={"form"} onSubmit={handleSubmit}>
                            <InputGroup sx={{
                                width: 400,
                            }} fullWidth={false} placeholder="Search for Logs" id="log" label="" validation={() => { return "" }} value={searchTerm} onChange={(e) => {
                                setSearchTerm(e.target.value)
                            }} />
                            <Button sx={{
                                height: "100%",
                                borderColor: neutral[800],
                            }} type="submit" variant="icon" aria-label='search for logs'>
                                <Search style={{ color: neutral[900] }} />
                            </Button>
                        </Box>
                    </Box>
                    <SimpleTable containerStyle={{ width: "100%" }} columns={mockColumns} dataSource={logs} elevated={false} />
                </Paper>
            </Box >
        </Box >
    )
}
