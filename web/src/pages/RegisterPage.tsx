import React, {useState} from "react";
import {Redirect} from "react-router";
import Page from "../components/Page";
import Input from "../ui/Input";
import Button from "../ui/Button";
import FormBlock from "../components/FormBlock";
import Field from "../ui/Field";
import {ErrorResp, registerUser} from "../api";
import Alert from "../ui/Alert";

const RegisterPage = () => {
	const [success, setSuccess] = useState<boolean>();
	const [error, setError] = useState<ErrorResp>({message: ""});
	const onSubmit = (event: any) => {
		event.preventDefault();
		const {login, password, email, first_name, last_name} = event.target;
		registerUser({
			login: login.value,
			password: password.value,
			email: email.value,
			first_name: first_name.value,
			last_name: last_name.value,
		})
			.then(() => {
				setError({message: ""});
				setSuccess(true);
			})
			.catch(error => {
				setError(error);
			});
	};
	if (success) {
		return <Redirect to={"/login"}/>
	}
	return <Page title="Register">
		<FormBlock onSubmit={onSubmit} title="Register" footer={
			<Button type="submit" color="primary">Register</Button>
		}>
			{error.message && <Alert>{error.message}</Alert>}
			<Field title="Username:" description={<>
				You can use only English letters, digits, symbols &laquo;<code>_</code>&raquo; and &laquo;<code>-</code>&raquo;.
				Username can starts only with English letter and ends with English letter and digit.
			</>}>
				<Input type="text" name="login" placeholder="Username" required autoFocus/>
				{error.invalid_fields && error.invalid_fields["login"] && <Alert>{error.invalid_fields["login"].message}</Alert>}
			</Field>
			<Field title="E-mail:" description="You will receive an email to verify your account.">
				<Input type="text" name="email" placeholder="E-mail" required/>
				{error.invalid_fields && error.invalid_fields["email"] && <Alert>{error.invalid_fields["email"].message}</Alert>}
			</Field>
			<Field title="Password:">
				<Input type="password" name="password" placeholder="Password" required/>
				{error.invalid_fields && error.invalid_fields["password"] && <Alert>{error.invalid_fields["password"].message}</Alert>}
			</Field>
			<Field title="Repeat password:">
				<Input type="password" name="password_repeat" placeholder="Repeat password" required/>
			</Field>
			<Field title="First name:">
				<Input type="text" name="first_name" placeholder="First name"/>
				{error.invalid_fields && error.invalid_fields["first_name"] && <Alert>{error.invalid_fields["first_name"].message}</Alert>}
			</Field>
			<Field title="Last name:">
				<Input type="text" name="last_name" placeholder="Last name"/>
				{error.invalid_fields && error.invalid_fields["last_name"] && <Alert>{error.invalid_fields["last_name"].message}</Alert>}
			</Field>
		</FormBlock>
	</Page>;
};

export default RegisterPage;
