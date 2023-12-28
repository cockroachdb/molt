import { useState } from 'react';
import {
    TextField,
    TextFieldProps
} from '@mui/material';


type InputTextProps = {
    id: string;
    value: any;
    onChange: (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => void;
    validation: (value: string) => string;
}

export type ValidatedTextFieldProps = InputTextProps & TextFieldProps;

export default function ValidatedTextField(props: ValidatedTextFieldProps) {
    const { id, value, onChange, validation, ...rest } = props;
    const [errorText, setErrorText] = useState("");

    return <TextField
        id={id}
        value={value}
        error={errorText !== ""}
        helperText={errorText}
        onChange={(e) => {
            onChange(e);
        }}
        onBlur={(_) => {
            const errText = validation(props.value);
            setErrorText(errText);
        }}
        size="small"
        {...rest}
    />
}