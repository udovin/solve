import React, {ReactNode} from "react";

const AuthContext = React.createContext(false);

class AuthProvider extends React.Component {
	state = {
		isAuth: false,
	};

	render(): ReactNode {
		const {children} = this.props;
		return (
			<AuthContext.Provider value={this.state.isAuth}>
				{children}
			</AuthContext.Provider>
		);
	}
}

const AuthConsumer = AuthContext.Consumer;

export {AuthProvider, AuthConsumer};
