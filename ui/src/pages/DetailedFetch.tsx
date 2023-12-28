import {
    Typography,
    Box,
    Button,
    Paper,
    Chip
} from '@mui/material';
import { useNavigate } from "react-router-dom";
import SimpleTable, { TableColumnProps } from '../components/tables/Table';
import { neutral } from '../styles/colors';
import { fontWeights } from '../styles/fonts';
import { DEFAULT_SPACING } from '../styles/theme';

export type LogLevel = "info" | "warning" | "danger";


export interface FetchStats {
    percentComplete: {
        description: string,
        data: number
    },
    numTables: {
        description: string,
        data: number
    },
    numRows: {
        description: string,
        data: number
    },
    numErrors: {
        description: string,
        data: number
    },
}

export interface FetchLogs {
    key: string;
    id: string;
    level: LogLevel;
    createdAt: string;
    message: string;
}

const mockStats: FetchStats = {
    percentComplete: {
        description: "Percentage Complete",
        data: 99
    },
    numTables: {
        description: "Number of Tables",
        data: 250
    },
    numRows: {
        description: "Number of Rows",
        data: 1000
    },
    numErrors: {
        description: "Number of Errors",
        data: 1
    },
}

const mockColumns: TableColumnProps<FetchLogs>[] = [
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
            return <Chip size="small" label={record.level.toUpperCase()} variant={record.level} />
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
const mockData: FetchLogs[] = [
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

export default function DetailedFetch() {
    const navigate = useNavigate();

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
                <Button onClick={() => navigate(-1)} sx={{ width: DEFAULT_SPACING * 2 }} variant="secondary" >Back</Button>
                <Typography sx={{ mb: 1 }} variant='h4'>Fetch Runs</Typography>
                <Paper sx={{
                    display: "flex",
                    flexDirection: "row",
                    justifyContent: "flex-start",
                    p: 2,
                }}>
                    {Object.keys(mockStats).map(key => {
                        const desc = mockStats[key as keyof typeof mockStats];
                        return <Box sx={{ borderRight: `1px solid ${neutral[400]}`, px: 2 }}>
                            <Typography fontWeight={fontWeights["heaviest"]} variant='body1'>
                                {desc.data}
                            </Typography>
                            <Typography fontWeight={fontWeights["light"]} variant='body2'>
                                {desc.description}
                            </Typography>
                        </Box>
                    })}
                </Paper>

                <SimpleTable containerStyle={{ width: "100%" }} columns={mockColumns} dataSource={mockData} />
            </Box>
        </Box>
    )
}
