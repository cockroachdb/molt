import {
    Typography,
    Box,
    FormControl,
    Select,
    MenuItem,
    Button,
    IconButton,
    InputLabel,
    SelectChangeEvent,
    SelectProps
} from '@mui/material';
import CloseIcon from '@material-ui/icons/Close';
import { FormEvent, useState } from 'react';

type LabelProps = {
    label: string;
    required?: boolean;
}
type SelectGroupProps = SelectProps<any> & LabelProps;

export default function SelectGroup(props: SelectGroupProps) {
    const { label, required, id, onChange, value, children, ...rest } = props;

    return (
        <FormControl fullWidth>
            <InputLabel id={`${id}-label`}
                required={required}
                shrink={false}
                sx={{
                    mt: -4,
                    ml: "-14px",
                    fontSize: "12px",
                }}>{label}</InputLabel>
            <Select
                labelId={`${id}-label`}
                id={id}
                size="small"
                value={value}
                onChange={onChange}
                {...rest}
            >
                {children}
            </Select>
        </FormControl>
    )
}
