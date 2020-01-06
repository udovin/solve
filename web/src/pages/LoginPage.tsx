import React, {useContext} from "react";
import Page from "../components/Page";
import Input from "../components/Input";
import Button from "../components/Button";
import FormBlock from "../components/FormBlock";
import {Redirect} from "react-router";
import {AuthContext} from "../AuthContext";
import Field from "../components/Field";
import {loginUser} from "../api";

const LoginPage = () => {
	const {session, setSession} = useContext(AuthContext);
	const onSubmit = (event: any) => {
		event.preventDefault();
		const {Login, Password} = event.target;
		loginUser({
			Login: Login.value,
			Password: Password.value,
		})
			.then(() => {
				fetch("/api/v0/sessions/current")
					.then(result => result.json())
					.then(result => setSession(result));
			});
	};
	if (session) {
		return <Redirect to={"/"}/>
	}
	return <Page title="Login">
		<FormBlock onSubmit={onSubmit} title="Login" footer={
			<Button type="submit" color="primary">Login</Button>
		}>
			<Field title="Username:">
				<Input type="text" name="Login" placeholder="Username" required autoFocus/>
			</Field>
			<Field title="Password:">
				<Input type="password" name="Password" placeholder="Password" required/>
			</Field>
		</FormBlock>
	</Page>;
};

export default LoginPage;
