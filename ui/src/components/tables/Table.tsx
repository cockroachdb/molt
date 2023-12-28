import {
    Table,
    TableProps,
    TableBody,
    TableCell,
    TableSortLabel,
    TableContainer,
    TableHead,
    TableRow,
    SxProps,
    Theme,
    Paper
} from '@mui/material';
import { useMemo, useState } from 'react';
import { fontWeights } from '../../styles/fonts';
import { neutral, primary } from '../../styles/colors';

type Order = "asc" | "desc";

function descendingComparator<T>(a: T, b: T, orderBy: keyof T) {
    if (b[orderBy] < a[orderBy]) {
        return -1;
    }
    if (b[orderBy] > a[orderBy]) {
        return 1;
    }
    return 0;
}

function getComparator<Key extends keyof any>(
    order: Order,
    orderBy: Key,
): (
    a: { [key in Key]: number | string },
    b: { [key in Key]: number | string },
) => number {
    return order === 'desc'
        ? (a, b) => descendingComparator(a, b, orderBy)
        : (a, b) => -descendingComparator(a, b, orderBy);
}

// TODO: figure out how to restrict sort directions
export interface TableColumnProps<T> {
    id: keyof T;
    title: React.ReactNode;
    sortDirections?: Order[];
    defaultSortOrder?: Order;
    width?: number | string;
    cellStyle?: SxProps<Theme>;
    align?: "left" | "center" | "right";
    render: (record: T, idx: number) => React.ReactNode;
}

type SimpleTableProps<T> = {
    columns: TableColumnProps<T>[];
    dataSource: T[];
    containerStyle?: SxProps<Theme>;
} & TableProps;

export interface DataSourceType {
    // key can be the same thing as the id.
    key: string;
    [x: string]: any;
}

function findFirstDefaultSortRuleColumn<T>(columns: TableColumnProps<T>[]) {
    return columns.find(item => item.defaultSortOrder !== undefined)
}

export default function SimpleTable<T extends DataSourceType>(props: SimpleTableProps<T>) {
    const { sx, containerStyle } = props;
    const defaultSortCol = findFirstDefaultSortRuleColumn(props.columns);
    const [order, setOrder] = useState<Order>(defaultSortCol?.defaultSortOrder || "asc");
    const [orderBy, setOrderBy] = useState<string>(defaultSortCol?.id as string || "");

    const handleRequestSort = (
        _: React.MouseEvent<unknown>,
        property: string,
    ) => {
        const isAsc = orderBy === property && order === 'asc';
        setOrder(isAsc ? 'desc' : 'asc');
        setOrderBy(property);
    };

    const createSortHandler =
        (property: string) => (event: React.MouseEvent<unknown>) => {
            handleRequestSort(event, property);
        };

    const visibleRows = useMemo(() =>
        (props.dataSource.sort(getComparator(order, orderBy || ""))), [props.dataSource, order, orderBy])

    return (
        <TableContainer sx={containerStyle} component={Paper}>
            <Table sx={{ minWidth: 650, ...sx }} aria-label="simple table">
                <TableHead>
                    <TableRow>
                        {props.columns.map(item => {
                            return <TableCell sx={item.cellStyle} align={item.align} key={item.id as string} sortDirection={"asc"}>
                                <TableSortLabel
                                    sx={{
                                        '& .MuiTableSortLabel-icon': {
                                            color: `${primary[3]} !important`,
                                        },
                                        color: neutral[900],
                                        fontWeight: fontWeights["heaviest"],
                                        textAlign: "center"
                                    }}
                                    onClick={createSortHandler(item.id as string)}
                                    direction={orderBy === item.id ? order : "asc"}
                                    active={orderBy === item.id}>
                                    {item.title}
                                </TableSortLabel>
                            </TableCell>
                        })}
                    </TableRow>
                </TableHead>
                <TableBody>
                    {visibleRows.map((row) => (
                        <TableRow
                            key={`datarow-${row.key}`}
                            sx={{ '&:last-child td, &:last-child th': { border: 0 } }}
                        >
                            {props.columns.map((item, idx) => {
                                return <TableCell align={item.align} key={`${row.key}-${item.id as string}`}>{item.render(row, idx)}</TableCell>
                            })}
                        </TableRow>
                    ))}
                </TableBody>
            </Table>
        </TableContainer >
    );
}