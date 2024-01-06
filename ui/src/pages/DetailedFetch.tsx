import { useState, useEffect, FormEvent } from 'react';
import { useParams } from 'react-router-dom';
import {
    Typography,
    Box,
    Button,
    Link,
    Paper,
    Chip,
    SelectChangeEvent,
    LinearProgress
} from '@mui/material';
import { Search, ChevronLeft } from '@material-ui/icons';
import { useNavigate, Link as RouterLink } from "react-router-dom";
import SimpleTable, { TableColumnProps } from '../components/tables/Table';
import { neutral, info } from '../styles/colors';
import { fontWeights } from '../styles/fonts';
import { DEFAULT_SPACING } from '../styles/theme';
import { InputGroup, Switch } from '../components';
import { createVerifyFromFetchTask, getSpecificFetchTask } from '../api';
import { FetchRun, FetchRunDetailed } from '../apigen';
import { formatNetDurationSeconds, formatSecondsToHHMMSS } from '../utils/dates';
import { Status, getChipFromStatus, getStatusFromString } from './FetchList';

const POLL_INTERVAL_MS = 1000;
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
    duration?: {
        description: string,
        data: string,
    },
}

export interface VerifyRun {
    key: string;
    id: string;
    name: string;
    status: Status;
    duration: string;
    startedAt: string;
    finishedAt: string;
    errors: number;
}

export interface FetchLog {
    key: string;
    id: string;
    level: LogLevel;
    createdAt: string;
    message: string;
}

const verifyColumns: TableColumnProps<VerifyRun>[] = [
    {
        id: "name",
        title: "Name",
        cellStyle: { width: "40%" },
        render: (record, _) => {
            return <Link sx={{
                color: info[3]
            }} underline='none' component={RouterLink} to={`/verify/${record.id}`}>{record.name}</Link>
        }
    },
    {
        id: "status",
        title: "Status",
        render: (record, _) => {
            return getChipFromStatus(record.status);
        }
    },
    {
        id: "startedAt",
        title: "Started At",
        render: (record, _) => {
            return record.startedAt;
        }
    },
];

const createColumns = (showPrettyPrint: boolean): TableColumnProps<FetchLog>[] => {
    return [
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
                if (showPrettyPrint) {
                    const prettyJSON = <pre>{JSON.stringify(JSON.parse(record.message), null, 2)}</pre>;
                    return prettyJSON;
                }

                return <Typography variant='body2'>{record.message}</Typography>;
            }
        },
    ];
}

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
    const [isLoading, setIsLoading] = useState(false);
    const [showPrettyPrint, setShowPrettyPrint] = useState(false);
    const [status, setStatus] = useState("");
    const [initialLogs, setInitialLogs] = useState<FetchLog[]>([]);
    const [verifyRuns, setVerifyRuns] = useState<VerifyRun[]>([]);
    const [fetchName, setFetchName] = useState(fetchId);
    const [logs, setLogs] = useState<FetchLog[]>([]);
    const [stats, setStats] = useState<FetchStats>({});
    const [isVerifyLoading, setIsVerifyLoading] = useState(false);
    const [verifyId, setVerifyId] = useState(0);

    const navigate = useNavigate();

    // TODO: need to add memoization and tune the interplay between the two useEffects
    // This is rendering more than it has to.
    // TODO: play around with interval timings for polling logs.
    // TODO: refactor this as a helper later on.
    useEffect(() => {
        const fetchData = async () => {
            try {
                const fid = Number(fetchId);
                const data = await getSpecificFetchTask(fid);

                if (data.status !== FetchRunDetailed.status.IN_PROGRESS) {
                    setIsLoading(false);
                }

                setFetchName(data.name);
                setStatus(data.status);

                const resLogs: FetchLog[] = data.logs.map(item => {
                    const createdAtTs = new Date(item.timestamp * 1000);

                    return {
                        key: `${item.timestamp}-${crypto.randomUUID()}`,
                        id: data.id.toString(),
                        level: getLevelFromString(item.level),
                        createdAt: createdAtTs.toISOString(),
                        message: item.message,
                    }
                });
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

                // Only put duration if it's relevant.
                const netDurStr = formatNetDurationSeconds(data.started_at, data.finished_at, data.stats?.net_duration_ms, data.stats?.export_duration_ms);
                if (netDurStr.trim() !== "") {
                    resStats.duration = {
                        description: "Net Duration",
                        data: netDurStr
                    }
                }

                setStats(resStats);

                // Set verify runs
                const mappedVerifyRuns: VerifyRun[] = data.verify_runs.map(item => {
                    const startedAtTs = new Date(item.started_at * 1000);
                    const finishedAtTs = new Date(item.finished_at * 1000);

                    return {
                        key: `${item.id.toString()}-${crypto.randomUUID()}`,
                        id: item.id.toString(),
                        name: item.name,
                        status: getStatusFromString(item.status),
                        duration: formatSecondsToHHMMSS(item.finished_at - item.started_at),
                        startedAt: startedAtTs.toISOString(),
                        finishedAt: finishedAtTs.toISOString(),
                        errors: 0,
                    }
                });
                setVerifyRuns(mappedVerifyRuns);
            } catch (e) {
                console.error(e);
            }
        }
        setIsLoading(true);
        fetchData()

        const interval = setInterval(() => {
            // Once the load finishes, stop polling.
            if (status !== FetchRun.status.IN_PROGRESS) {
                clearInterval(interval);
                return;
            }

            fetchData()
        }, POLL_INTERVAL_MS)
        return () => {
            clearInterval(interval);
        }
    }, [fetchId, status, stats.percentComplete?.data]);

    // Handling the verify fetch data.
    useEffect(() => {
        const fetchData = async () => {
            try {
                const fid = Number(fetchId);
                const data = await getSpecificFetchTask(fid);

                // Set verify runs
                const mappedVerifyRuns: VerifyRun[] = data.verify_runs.map(item => {
                    const startedAtTs = new Date(item.started_at * 1000);
                    const finishedAtTs = new Date(item.finished_at * 1000);

                    return {
                        key: `${item.id.toString()}-${crypto.randomUUID()}`,
                        id: item.id.toString(),
                        name: item.name,
                        status: getStatusFromString(item.status),
                        duration: formatSecondsToHHMMSS(item.finished_at - item.started_at),
                        startedAt: startedAtTs.toISOString(),
                        finishedAt: finishedAtTs.toISOString(),
                        errors: 0,
                    }
                });
                setVerifyRuns(mappedVerifyRuns);
            } catch (e) {
                console.error(e);
            }
        }

        const interval = setInterval(() => {
            // Once the load finishes, stop polling.
            if (verifyId === 0 || verifyRuns.find(item => Number(item.id) === verifyId)) {
                clearInterval(interval);
                setIsVerifyLoading(false);
                return;
            }

            fetchData()
        }, POLL_INTERVAL_MS)

        return () => {
            clearInterval(interval);
        }
    }, [fetchId, verifyId, verifyRuns])

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
                <Box sx={{
                    display: "flex",
                    flexDirection: "row",
                    alignItems: "center",
                    gap: 2,
                }}>
                    <Typography sx={{ mb: 1 }} variant='h4'>{fetchName}</Typography>
                    <Chip sx={{ width: 120 }} size="medium" variant={status === FetchRunDetailed.status.SUCCESS ? "success" : status === FetchRunDetailed.status.FAILURE ? "danger" : "info"} label={status} />
                </Box>
                {isLoading && <LinearProgress />}
                {!isLoading && <Paper sx={{ p: 2, }}>
                    <Typography sx={{ mb: 1 }} variant='h6'>Stats</Typography>
                    <Box sx={{
                        display: "flex",
                        flexDirection: "row",
                        justifyContent: "flex-start",
                    }}>
                        {Object.keys(stats).map(key => {
                            const desc = stats[key as keyof typeof stats];
                            return <Box key={key} sx={{ borderLeft: `1px solid ${neutral[400]}`, px: 2 }}>
                                <Typography color="primary" fontWeight={fontWeights["heaviest"]} variant='body1'>
                                    {desc?.data}
                                </Typography>
                                <Typography fontWeight={fontWeights["light"]} variant='body2'>
                                    {desc?.description}
                                </Typography>
                            </Box>
                        })}
                    </Box>
                </Paper>}
                <Paper sx={{
                    p: 2
                }}>
                    <Box sx={{
                        display: "flex",
                        flexDirection: "row",
                        alignItems: "center",
                        justifyContent: "space-between",
                    }}>
                        <Typography sx={{ mb: 1 }} variant='h6'>Verify Runs</Typography>
                        {status === FetchRunDetailed.status.SUCCESS && <Button variant="contained" onClick={async () => {
                            try {
                                const vId = await createVerifyFromFetchTask(Number(fetchId));
                                setVerifyId(vId);
                                setIsVerifyLoading(true);
                            } catch (e) {
                                console.error(e);
                            }
                        }}>Run Verify</Button>}
                    </Box>
                    {isVerifyLoading && <LinearProgress />}
                    <SimpleTable containerStyle={{ width: "100%" }} columns={verifyColumns} dataSource={verifyRuns} elevated={false} />
                </Paper>
                <Paper sx={{
                    p: 2
                }}>
                    <Typography sx={{ mb: 1 }} variant='h6'>Logs</Typography>
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
                    <Switch
                        sx={{
                            my: 2
                        }}
                        required
                        label="Pretty print logs?"
                        id="prettyPrint"
                        value={showPrettyPrint}
                        onChange={(_: SelectChangeEvent) => {
                            setShowPrettyPrint(!showPrettyPrint);
                        }}
                    />
                    <SimpleTable containerStyle={{ width: "100%" }} columns={createColumns(showPrettyPrint)} dataSource={logs} elevated={false} />
                </Paper>
            </Box >
        </Box >
    )
}
