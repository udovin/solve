import React, {ReactNode} from "react";
import {CurrentSession} from "./api";

export interface AuthState {
	session: CurrentSession | null;
}

const AuthContext = React.createContext<AuthState>({session: null});

class AuthProvider extends React.Component {
	state: AuthState = {session: null};

	componentDidMount(): void {
		let request = new XMLHttpRequest();
		request.open("GET", "/api/v0/sessions/current", true);
		request.setRequestHeader("Content-Type", "application/json; charset=UTF-8");
		request.responseType = "json";
		let that = this;
		request.onload = function() {
			if (this.status !== 200) {
				that.setState({
					session: null,
				})
			} else {
				that.setState({
					session: this.response,
				})
			}
		};
		request.send();
	}

	render(): ReactNode {
		const {children} = this.props;
		return (
			<AuthContext.Provider value={this.state}>
				{children}
			</AuthContext.Provider>
		);
	}
}

const AuthConsumer = AuthContext.Consumer;

export {AuthProvider, AuthConsumer};
