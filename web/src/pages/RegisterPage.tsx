import React, {useState} from "react";
import Page from "../layout/Page";
import Input from "../layout/Input";
import Button from "../layout/Button";
import {FormBlock} from "../layout/blocks";
import {Redirect} from "react-router";
import Field from "../layout/Field";

const RegisterPage = () => {
	const [success, setSuccess] = useState<boolean>();
	const onSubmit = (event: any) => {
		event.preventDefault();
		const {login, password, email} = event.target;
		fetch("/api/v0/users", {
			method: "POST",
			headers: {
				"Content-Type": "application/json; charset=UTF-8",
			},
			body: JSON.stringify({
				Login: login.value,
				Password: password.value,
				Email: email.value,
			})
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
				<Input type="text" name="login" placeholder="Username" required autoFocus/>
			</Field>
			<Field title="E-mail:" description="You will receive an email to verify your account.">
				<Input type="text" name="email" placeholder="E-mail" required/>
			</Field>
			<Field title="Password:">
				<Input type="password" name="password" placeholder="Password" required/>
			</Field>
			<Field title="Repeat password:">
				<Input type="password" name="passwordRepeat" placeholder="Repeat password" required/>
			</Field>
		</FormBlock>
	</Page>;
};

export default RegisterPage;
