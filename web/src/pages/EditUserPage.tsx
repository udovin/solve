import React, {useEffect, useState} from "react";
import {Redirect, RouteComponentProps} from "react-router";
import Page from "../layout/Page";
import {User} from "../api";
import {FormBlock} from "../layout/blocks";
import "./ContestPage.scss"
import Input from "../layout/Input";
import {Button} from "../layout/buttons";

type UserPageParams = {
	UserID: string;
}

const EditUserPage = ({match}: RouteComponentProps<UserPageParams>) => {
	const {UserID} = match.params;
	const [user, setUser] = useState<User>();
	const [success, setSuccess] = useState<boolean>();
	useEffect(() => {
		fetch("/api/v0/users/" + UserID)
			.then(result => result.json())
			.then(result => setUser(result));
	}, [UserID]);
	if (!user) {
		return <>Loading...</>;
	}
	const onSubmit = (event: any) => {
		event.preventDefault();
		const {password, passwordRepeat} = event.target;
		if (password.value.length < 8 || password.value.length > 32 ||
			password !== passwordRepeat) {
			setSuccess(false);
			return;
		}
		fetch("/api/v0/users/" + user.ID, {
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
		return <Redirect to={"/users/" + UserID} push={true}/>
	}
	const {Login} = user;
	return <Page title={Login}>
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
	</Page>;
};

export default EditUserPage;
