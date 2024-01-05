import { useEffect, useState } from 'react';
import {
    Typography,
    Box,
    Link,
    Chip,
    Button,
} from '@mui/material';
import { Link as RouterLink, useNavigate } from "react-router-dom";
import SimpleTable, { TableColumnProps } from '../components/tables/Table';
import { info } from '../styles/colors';
import { SETUP_CONNECTION_PATH } from '.';
import { getFetchTasks } from '../api';
import { formatSecondsToHHMMSS } from '../utils/dates';


export type Status = "In Progress" | "Ready for Review" | "Succeeded" | "Failed" | "Unknown"

export interface FetchRun {
    key: string;
    id: string;
    name: string;
    status: Status;
    duration: string;
    startedAt: string;
    finishedAt: string;
    errors: number;
}

export const getChipFromStatus = (status: Status) => {
    switch (status) {
        case "Succeeded":
            return <Chip size="small" label={status} variant="success" />
        case "Failed":
            return <Chip size="small" label={status} variant="danger" />
        case "Ready for Review":
            return <Chip size="small" label={status} variant="warning" />
    }

    return <Chip size="small" label={status} variant="info" />
}

const columns: TableColumnProps<FetchRun>[] = [
    {
        id: "name",
        title: "Name",
        cellStyle: { width: "40%" },
        render: (record, _) => {
            return <Link sx={{
                color: info[3]
            }} underline='none' component={RouterLink} to={`/fetch/${record.id}`}>{record.name}</Link>
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
        id: "duration",
        title: "Duration",
        render: (record, _) => {
            return record.duration
        }
    },
    {
        id: "startedAt",
        title: "Started At",
        render: (record, _) => {
            return record.startedAt;
        }
    },
    {
        id: "finishedAt",
        title: "Finished At",
        render: (record, _) => {
            return record.finishedAt;
        }
    },
    {
        id: "errors",
        title: "Errors",
        render: (record, _) => {
            return record.errors;
        }
    },
];

export const getStatusFromString = (input: string): Status => {
    switch (input) {
        case "IN_PROGRESS":
            return "In Progress"
        case "SUCCESS":
            return "Succeeded"
        case "FAILURE":
            return "Failed"
    }

    return "Unknown"
}

export default function FetchList() {
    const navigate = useNavigate();
    const [runs, setRuns] = useState<FetchRun[]>([]);

    // TODO: refactor this as a helper later on.
    useEffect(() => {
        const fetchData = async () => {
            try {
                const data = await getFetchTasks();

                const mappedRuns: FetchRun[] = data.map(item => {
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
                })

                setRuns(mappedRuns);
            } catch (e) {
                console.error(e);
            }
        }
        fetchData()
    }, [])

    return (
        <Box sx={{
            display: "flex",
            flexDirection: "column",
            width: "80%",
            gap: 2,
            py: 4,
            px: 2
        }}>
            <Typography sx={{ mb: 1 }} variant='h4'>Fetch Runs</Typography>
            <Button sx={{ width: "120px", alignSelf: "flex-end" }} fullWidth={false} variant="contained"
                onClick={() => {
                    navigate(SETUP_CONNECTION_PATH);
                }}>Create New</Button>
            <SimpleTable columns={columns} dataSource={runs} />
        </Box>
    )
}
