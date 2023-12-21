import {
    FormControl,
    InputLabel,
} from '@mui/material';
import ValidatedTextField, { ValidatedTextFieldProps } from './ValidatedTextField';

type LabelProps = {
    label: string;
    required?: boolean;
}
type InputGroupProps = ValidatedTextFieldProps & LabelProps;

export default function InputGroup(props: InputGroupProps) {
    const { label, required, id, value, validation, onChange, ...rest } = props;
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
            <ValidatedTextField
                id={id}
                value={value}
                validation={validation}
                onChange={onChange}
                {...rest} />
        </FormControl>
    )
}
