import React, {useState} from "react";
import {Redirect} from "react-router";
import Page from "../components/Page";
import Input from "../components/Input";
import Button from "../components/Button";
import FormBlock from "../components/FormBlock";
import Field from "../components/Field";
import {registerUser} from "../api";

const RegisterPage = () => {
	const [success, setSuccess] = useState<boolean>();
	const onSubmit = (event: any) => {
		event.preventDefault();
		const {Login, Password, Email, FirstName, LastName} = event.target;
		registerUser({
			Login: Login.value,
			Password: Password.value,
			Email: Email.value,
			FirstName: FirstName.value,
			LastName: LastName.value,
		})
			.then(() => setSuccess(true));
	};
	if (success) {
		return <Redirect to={"/login"}/>
	}
	return <Page title="Register">
		<FormBlock onSubmit={onSubmit} title="Register" footer={
			<Button type="submit" color="primary">Register</Button>
		}>
			<Field title="Username:" description={<>
				You can use only English letters, digits, symbols &laquo;<code>_</code>&raquo; and &laquo;<code>-</code>&raquo;.
				Username can starts only with English letter and ends with English letter and digit.
			</>}>
				<Input type="text" name="Login" placeholder="Username" required autoFocus/>
			</Field>
			<Field title="E-mail:" description="You will receive an email to verify your account.">
				<Input type="text" name="Email" placeholder="E-mail" required/>
			</Field>
			<Field title="Password:">
				<Input type="password" name="Password" placeholder="Password" required/>
			</Field>
			<Field title="Repeat password:">
				<Input type="password" name="PasswordRepeat" placeholder="Repeat password" required/>
			</Field>
			<Field title="First name:">
				<Input type="text" name="FirstName" placeholder="First name"/>
			</Field>
			<Field title="Last name:">
				<Input type="text" name="LastName" placeholder="Last name"/>
			</Field>
		</FormBlock>
	</Page>;
};

export default RegisterPage;
