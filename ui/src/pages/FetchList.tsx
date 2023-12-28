import {
    Typography,
    Box,
    Link,
    Chip
} from '@mui/material';
import { Link as RouterLink } from "react-router-dom";
import SimpleTable, { TableColumnProps } from '../components/tables/Table';
import { info } from '../styles/colors';

type FetchStatus = "In Progress" | "Ready for Review" | "Completed" | "Failed"

interface FetchRun {
    key: string;
    id: string;
    name: string;
    status: FetchStatus;
    startedAt: string;
    updatedAt: string;
    errors: number;
}

const getChipFromStatus = (status: FetchStatus) => {
    switch (status) {
        case "Completed":
            return <Chip size="small" label={status} variant="success" />
        case "Failed":
            return <Chip size="small" label={status} variant="danger" />
        case "Ready for Review":
            return <Chip size="small" label={status} variant="warn" />
    }

    return <Chip size="small" label={status} variant="info" />
}

const mockColumns: TableColumnProps<FetchRun>[] = [
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
        id: "startedAt",
        title: "Started At",
        render: (record, _) => {
            return record.startedAt;
        }
    },
    {
        id: "updatedAt",
        title: "Updated At",
        render: (record, _) => {
            return record.updatedAt;
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
const mockData: FetchRun[] = [
    {
        key: 'run1',
        id: "1",
        name: 'Run 1',
        status: "Completed",
        startedAt: '2023-01-02T10:00:00Z',
        updatedAt: '2023-01-02T11:30:00Z',
        errors: 0,
    },
    {
        key: 'run2',
        id: "2",
        name: 'Run 2',
        status: "Failed",
        startedAt: '2023-01-03T15:45:00Z',
        updatedAt: '2023-01-03T16:30:00Z',
        errors: 2,
    },
    {
        key: 'run3',
        id: "3",
        name: 'Run 3',
        status: "In Progress",
        startedAt: '2023-01-03T15:45:00Z',
        updatedAt: '2023-01-03T16:30:00Z',
        errors: 5,
    },
    {
        key: 'run4',
        id: "4",
        name: 'Run 4',
        status: "Ready for Review",
        startedAt: '2023-01-03T15:45:00Z',
        updatedAt: '2023-01-03T16:30:00Z',
        errors: 7,
    },
];

export default function FetchList() {
    return (
        <Box sx={{
            display: "flex",
            flexDirection: "column",
            gap: 2,
            py: 4,
            px: 2
        }}>
            <Typography sx={{ mb: 1 }} variant='h4'>Fetch Runs</Typography>
            <SimpleTable containerStyle={{ width: "80%" }} columns={mockColumns} dataSource={mockData} />
        </Box>
    )
}
