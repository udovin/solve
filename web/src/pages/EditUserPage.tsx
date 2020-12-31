import React, {useContext, useEffect, useState} from "react";
import {Redirect, RouteComponentProps} from "react-router";
import Page from "../components/Page";
import {Session, User} from "../api";
import FormBlock from "../components/FormBlock";
import Input from "../ui/Input";
import Button from "../ui/Button";
import SessionsBlock from "../components/SessionsBlock";
import {AuthContext} from "../AuthContext";

type UserPageParams = {
	user_id: string;
}

const EditUserPage = ({match}: RouteComponentProps<UserPageParams>) => {
	const {user_id} = match.params;
	const [user, setUser] = useState<User>();
	const [sessions, setSessions] = useState<Session[]>();
	const {status} = useContext(AuthContext);
	const [success, setSuccess] = useState<boolean>();
	useEffect(() => {
		fetch(`/api/v0/users/${user_id}`)
			.then(result => result.json())
			.then(result => setUser(result));
	}, [user_id]);
	useEffect(() => {
		fetch(`/api/v0/users/${user_id}/sessions`)
			.then(result => result.json())
			.then(result => setSessions(result));
	}, [user_id]);
	if (!status || !user) {
		return <>Loading...</>;
	}
	const onSubmit = (event: any) => {
		event.preventDefault();
		const {password, passwordRepeat} = event.target;
		if (password.value.length < 8 || password.value.length > 32 ||
			password.value !== passwordRepeat.value) {
			setSuccess(false);
			return;
		}
		fetch("/api/v0/users/" + user.id, {
			method: "PATCH",
			headers: {
				"Content-Type": "application/json; charset=UTF-8",
			},
			body: JSON.stringify({
				Password: password.value,
			})
		})
			.then(() => setSuccess(true));
	};
	if (success) {
		return <Redirect to={"/users/" + user_id} push={true}/>
	}
	const {login} = user;
	return <Page title={login}>
		<FormBlock title="Change password" onSubmit={onSubmit} footer={
			<Button type="submit">Change</Button>
		}>
			<div className="ui-field">
				<label>
					<span className="label">New password:</span>
					<Input type="password" name="password" placeholder="New password" required autoFocus/>
				</label>
			</div>
			<div className="ui-field">
				<label>
					<span className="label">Repeat new password:</span>
					<Input type="password" name="passwordRepeat" placeholder="Repeat new password" required/>
				</label>
			</div>
		</FormBlock>
		{sessions ?
			<SessionsBlock sessions={sessions} currentSession={status.session}/> :
			<>Loading...</>}
	</Page>;
};

export default EditUserPage;
