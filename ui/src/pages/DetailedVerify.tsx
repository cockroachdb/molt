import { useState, useEffect, FormEvent } from 'react';
import { useParams } from 'react-router-dom';
import {
    Typography,
    Box,
    Button,
    Paper,
    Chip,
    SelectChangeEvent,
    LinearProgress
} from '@mui/material';
import { Search, ChevronLeft } from '@material-ui/icons';
import { useNavigate } from "react-router-dom";
import SimpleTable, { TableColumnProps } from '../components/tables/Table';
import { neutral } from '../styles/colors';
import { fontSizes, fontWeights } from '../styles/fonts';
import { DEFAULT_SPACING } from '../styles/theme';
import { InputGroup, Switch } from '../components';
import { getSpecificVerifyTask } from '../api';
import { VerifyRunDetailed } from '../apigen';
import { formatNetDurationSeconds, } from '../utils/dates';
import { Status } from './FetchList';

const POLL_INTERVAL_MS = 1000;
export type LogLevel = "info" | "warning" | "danger";

export interface VerifyStats {
    numTables?: {
        description: string,
        data: number
    },
    numRows?: {
        description: string,
        data: number
    },
    numSuccess?: {
        description: string,
        data: number
    },
    numConditionalSuccess?: {
        description: string,
        data: number
    },
    numColumnMismatch?: {
        description: string,
        data: number
    },
    numExtraneous?: {
        description: string,
        data: number
    },
    numLiveRetry?: {
        description: string,
        data: number
    },
    numMismatch?: {
        description: string,
        data: number
    },
    numMissing?: {
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

export interface VerifyMismatch {
    key: string;
    id: string;
    level: LogLevel;
    createdAt: string;
    message: string;
    category: string;
    primaryKey: string;
    sourceValues: string;
    targetValues: string;
    table: string;
}

const createMismatchColumns = (showPrettyPrint: boolean): TableColumnProps<VerifyMismatch>[] => {
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
            id: "table",
            title: "Table",
            cellStyle: { width: "15%" },
            render: (record, _) => {
                return <Typography variant='body2'>{record.table}</Typography>;
            }
        },
        {
            id: "category",
            title: "Category",
            cellStyle: { width: "15%" },
            render: (record, _) => {
                return <Chip size="small" label={record.category} variant={record.level} />
            }
        },
        {
            id: "message",
            title: "Message",
            cellStyle: { width: "30%" },
            render: (record, _) => {
                if (showPrettyPrint && record.message !== undefined) {
                    const prettyJSON = <pre>{JSON.stringify(JSON.parse(record.message), null, 2)}</pre>;
                    return prettyJSON;
                }

                return <Typography variant='body2'>{record.message}</Typography>;
            }
        },
        {
            id: "primaryKey",
            title: "Primary Key",
            cellStyle: { width: "70%" },
            render: (record, _) => {
                return <Typography variant='body2'>{record.primaryKey}</Typography>;
            }
        },
        {
            id: "sourceValues",
            title: "Source Values",
            cellStyle: { width: "70%" },
            render: (record, _) => {
                if (showPrettyPrint && record.sourceValues !== undefined) {
                    const prettyJSON = <pre>{JSON.stringify(JSON.parse(record.sourceValues), null, 2)}</pre>;
                    return prettyJSON;
                }

                return <Typography variant='body2'>{record.sourceValues}</Typography>;
            }
        },
        {
            id: "targetValues",
            title: "Target Values",
            cellStyle: { width: "70%" },
            render: (record, _) => {
                if (showPrettyPrint && record.targetValues !== undefined) {
                    const prettyJSON = <pre>{JSON.stringify(JSON.parse(record.targetValues), null, 2)}</pre>;
                    return prettyJSON;
                }

                return <Typography variant='body2'>{record.targetValues}</Typography>;
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

export default function DetailedVerify() {
    const { verifyId } = useParams();
    const [searchTerm, setSearchTerm] = useState("");
    const [isLoading, setIsLoading] = useState(false);
    const [showPrettyPrint, setShowPrettyPrint] = useState(false);
    const [status, setStatus] = useState("");
    const [initialMismatches, setInitialMismatches] = useState<VerifyMismatch[]>([]);
    const [name, setName] = useState(verifyId);
    const [mismatches, setMismatches] = useState<VerifyMismatch[]>([]);
    const [stats, setStats] = useState<VerifyStats>({});

    const navigate = useNavigate();

    // TODO: refactor this as a helper later on.
    useEffect(() => {
        const getData = async () => {
            try {
                const vid = Number(verifyId);
                const data = await getSpecificVerifyTask(vid);

                if (data.status !== VerifyRunDetailed.status.IN_PROGRESS) {
                    setIsLoading(false);
                }

                setName(data.name);
                setStatus(data.status);


                const resLogs: VerifyMismatch[] = data.mismatches.map(item => {
                    const createdAtTs = new Date(item.timestamp * 1000);
                    const jsonMessage = JSON.parse(item.message)

                    return {
                        key: `${item.timestamp}-${crypto.randomUUID()}`,
                        id: data.id.toString(),
                        level: getLevelFromString(item.level),
                        createdAt: createdAtTs.toISOString(),
                        message: item.message,
                        category: item.type,
                        primaryKey: jsonMessage["primary_key"],
                        sourceValues: JSON.stringify(jsonMessage["source_values"]),
                        targetValues: JSON.stringify(jsonMessage["target_values"]),
                        table: `${item.schema}.${item.table}`
                    }
                });
                setMismatches(resLogs);
                setInitialMismatches(resLogs);


                const resStats: VerifyStats = {
                    numTables: {
                        description: "Tables",
                        data: data.stats.num_tables,
                    },
                    numRows: {
                        description: "Rows",
                        data: data.stats?.num_truth_rows
                    },
                    numSuccess: {
                        description: "Success",
                        data: data.stats?.num_success,
                    },
                    numConditionalSuccess: {
                        description: "Conditional Success",
                        data: data.stats?.num_conditional_success,
                    },
                    numColumnMismatch: {
                        description: "Column Mismatch",
                        data: data.stats?.num_column_mismatch
                    },
                    numExtraneous: {
                        description: "Extraneous",
                        data: data.stats?.num_extraneous
                    },
                    numMismatch: {
                        description: "Row Mismatching",
                        data: data.stats?.num_mismatch
                    },
                    numMissing: {
                        description: "Row Missing",
                        data: data.stats?.num_missing
                    },
                }

                // Only put duration if it's relevant.
                const netDurStr = formatNetDurationSeconds(data.started_at, data.finished_at, data.stats?.net_duration_ms, data.stats?.net_duration_ms);
                if (netDurStr.trim() !== "") {
                    resStats.duration = {
                        description: "Net Duration",
                        data: netDurStr
                    }
                }

                setStats(resStats);
            } catch (e) {
                console.error(e);
            }
        }
        setIsLoading(true);
        getData()

        const interval = setInterval(() => {
            // Once the load finishes, stop polling.
            if (status !== VerifyRunDetailed.status.IN_PROGRESS) {
                clearInterval(interval);
                return;
            }

            getData()
        }, POLL_INTERVAL_MS)
        return () => {
            clearInterval(interval);
        }
    }, [verifyId, status]);

    const handleSubmit = (e: FormEvent) => {
        e.preventDefault();

        if (searchTerm === "") {
            setMismatches(initialMismatches);
            return;
        }

        const filteredData = mismatches.filter(item => item.message.includes(searchTerm));
        setMismatches(filteredData);
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
                    <Typography sx={{ mb: 1 }} variant='h4'>{name}</Typography>
                    <Chip sx={{ width: 120 }} size="medium" variant={status === VerifyRunDetailed.status.SUCCESS ? "success" : status === VerifyRunDetailed.status.FAILURE ? "danger" : "info"} label={status} />
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

                            if (desc?.data === "" || desc?.data === 0) {
                                return undefined;
                            }

                            return <Box key={key} sx={{ borderLeft: `1px solid ${neutral[400]}`, px: 2 }}>
                                <Typography color="primary" fontWeight={fontWeights["heaviest"]} variant='body2'>
                                    {desc?.data}
                                </Typography>
                                <Typography fontSize={fontSizes["md"]} fontWeight={fontWeights["light"]}>
                                    {desc?.description}
                                </Typography>
                            </Box>
                        })}
                    </Box>
                </Paper>}
                <Paper sx={{
                    p: 2
                }}>
                    <Typography sx={{ mb: 1 }} variant='h6'>Mismatches</Typography>
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
                    <SimpleTable containerStyle={{ width: "100%" }} columns={createMismatchColumns(showPrettyPrint)} dataSource={mismatches} elevated={false} />
                </Paper>
            </Box >
        </Box >
    )
}
