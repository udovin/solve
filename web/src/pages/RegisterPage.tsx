import React, {useState} from "react";
import Page from "../layout/Page";
import Input from "../layout/Input";
import {Button} from "../layout/buttons";
import {FormBlock} from "../layout/blocks";
import {Redirect} from "react-router";

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
			<div className="ui-field">
				<label>
					<span className="label">Username:</span>
					<Input type="text" name="login" placeholder="Username" required autoFocus/>
				</label>
				<span className="text">
					You can use only English letters, digits, symbols &laquo;<code>_</code>&raquo; and &laquo;<code>-</code>&raquo;.
					Username can starts only with English letter and ends with English letter and digit.
				</span>
			</div>
			<div className="ui-field">
				<label>
					<span className="label">E-mail:</span>
					<Input type="text" name="email" placeholder="E-mail" required/>
				</label>
				<span className="text">
					You will receive an email to verify your account.
				</span>
			</div>
			<div className="ui-field">
				<label>
					<span className="label">Password:</span>
					<Input type="password" name="password" placeholder="Password" required/>
				</label>
			</div>
			<div className="ui-field">
				<label>
					<span className="label">Repeat password:</span>
					<Input type="password" name="passwordRepeat" placeholder="Repeat password" required/>
				</label>
			</div>
		</FormBlock>
	</Page>;
};

export default RegisterPage;
