export namespace model {
	
	export class TodayView {
	    checkedIn: boolean;
	    time: string;
	    credit?: number;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new TodayView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.checkedIn = source["checkedIn"];
	        this.time = source["time"];
	        this.credit = source["credit"];
	        this.message = source["message"];
	    }
	}
	export class AccountView {
	    id: string;
	    nickname: string;
	    email: string;
	    status: string;
	    tokenSuffix: string;
	    expiresAt: number;
	    isActive: boolean;
	    today: TodayView;
	    creditBalance?: number;
	    creditBalanceTotal?: number;
	    creditBalanceAt: number;
	
	    static createFrom(source: any = {}) {
	        return new AccountView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.nickname = source["nickname"];
	        this.email = source["email"];
	        this.status = source["status"];
	        this.tokenSuffix = source["tokenSuffix"];
	        this.expiresAt = source["expiresAt"];
	        this.isActive = source["isActive"];
	        this.today = this.convertValues(source["today"], TodayView);
	        this.creditBalance = source["creditBalance"];
	        this.creditBalanceTotal = source["creditBalanceTotal"];
	        this.creditBalanceAt = source["creditBalanceAt"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class CheckinResult {
	    id: string;
	    success: boolean;
	    message: string;
	    credit?: number;
	    checkedInAt: number;
	
	    static createFrom(source: any = {}) {
	        return new CheckinResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.success = source["success"];
	        this.message = source["message"];
	        this.credit = source["credit"];
	        this.checkedInAt = source["checkedInAt"];
	    }
	}
	export class CheckinSummary {
	    checkedIn: number;
	    total: number;
	    reloginRequired: number;
	
	    static createFrom(source: any = {}) {
	        return new CheckinSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.checkedIn = source["checkedIn"];
	        this.total = source["total"];
	        this.reloginRequired = source["reloginRequired"];
	    }
	}
	export class LoginStart {
	    authUrl: string;
	    expiresIn: number;
	
	    static createFrom(source: any = {}) {
	        return new LoginStart(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.authUrl = source["authUrl"];
	        this.expiresIn = source["expiresIn"];
	    }
	}
	export class LoginStatus {
	    stage: string;
	    done: boolean;
	    error: string;
	    account?: AccountView;
	
	    static createFrom(source: any = {}) {
	        return new LoginStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.stage = source["stage"];
	        this.done = source["done"];
	        this.error = source["error"];
	        this.account = this.convertValues(source["account"], AccountView);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class PendingUpdateInfo {
	    exists: boolean;
	    version: string;
	
	    static createFrom(source: any = {}) {
	        return new PendingUpdateInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.exists = source["exists"];
	        this.version = source["version"];
	    }
	}
	export class QuotaView {
	    balance?: number;
	    total?: number;
	    at: number;
	
	    static createFrom(source: any = {}) {
	        return new QuotaView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.balance = source["balance"];
	        this.total = source["total"];
	        this.at = source["at"];
	    }
	}
	export class Settings {
	    autoStart: boolean;
	    updateProxy: string;
	    proxyEnabled: boolean;
	    proxyPort: number;
	    activeCredentialId: string;
	    telemetryEnabled: boolean;
	    telemetryId: string;
	    telemetryLastAt: number;
	
	    static createFrom(source: any = {}) {
	        return new Settings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.autoStart = source["autoStart"];
	        this.updateProxy = source["updateProxy"];
	        this.proxyEnabled = source["proxyEnabled"];
	        this.proxyPort = source["proxyPort"];
	        this.activeCredentialId = source["activeCredentialId"];
	        this.telemetryEnabled = source["telemetryEnabled"];
	        this.telemetryId = source["telemetryId"];
	        this.telemetryLastAt = source["telemetryLastAt"];
	    }
	}
	
	export class UpdateCheckResult {
	    status: string;
	    currentVersion: string;
	    latestVersion: string;
	    releaseUrl: string;
	    downloadUrl: string;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateCheckResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.currentVersion = source["currentVersion"];
	        this.latestVersion = source["latestVersion"];
	        this.releaseUrl = source["releaseUrl"];
	        this.downloadUrl = source["downloadUrl"];
	        this.message = source["message"];
	    }
	}
	export class UpdateDownloadEvent {
	    phase: string;
	    downloaded: number;
	    total: number;
	    percent: number;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateDownloadEvent(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.phase = source["phase"];
	        this.downloaded = source["downloaded"];
	        this.total = source["total"];
	        this.percent = source["percent"];
	        this.message = source["message"];
	    }
	}

}

export namespace proxy {
	
	export class Status {
	    running: boolean;
	    port: number;
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new Status(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.running = source["running"];
	        this.port = source["port"];
	        this.error = source["error"];
	    }
	}

}

