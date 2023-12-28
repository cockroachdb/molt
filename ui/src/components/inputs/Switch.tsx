import {
    FormControl,
    FormControlLabel,
    Switch as SwitchBase,
    SwitchProps as SwitchBaseProps
} from '@mui/material';

type LabelProps = {
    label: string;
    required?: boolean;
}
type InputGroupProps = SwitchBaseProps & LabelProps;

export default function Switch(props: InputGroupProps) {
    const { label, required, id, value, onChange, ...rest } = props;
    return (
        <FormControl fullWidth>
            <FormControlLabel
                sx={{ m: 0, gap: 1 }}
                label={label}
                control={<SwitchBase id={id}
                    value={value}
                    onChange={onChange}
                    {...rest} />} />
        </FormControl>
    )
}
