export namespace spgui {
	
	export class Config {
	    ConfigPath: string;
	    SiteURL: string;
	    GlobalTimeoutSec: number;
	    CleanOutput: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ConfigPath = source["ConfigPath"];
	        this.SiteURL = source["SiteURL"];
	        this.GlobalTimeoutSec = source["GlobalTimeoutSec"];
	        this.CleanOutput = source["CleanOutput"];
	    }
	}
	export class ListQuery {
	    list: string;
	    select: string;
	    filter: string;
	    orderby: string;
	    top: number;
	    all: boolean;
	    latestOnly: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ListQuery(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.list = source["list"];
	        this.select = source["select"];
	        this.filter = source["filter"];
	        this.orderby = source["orderby"];
	        this.top = source["top"];
	        this.all = source["all"];
	        this.latestOnly = source["latestOnly"];
	    }
	}
	export class QuerySummary {
	    items: number;
	    pages: number;
	    throttled: boolean;
	    partial: boolean;
	    fallback: boolean;
	    stoppedEarly: boolean;
	
	    static createFrom(source: any = {}) {
	        return new QuerySummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.items = source["items"];
	        this.pages = source["pages"];
	        this.throttled = source["throttled"];
	        this.partial = source["partial"];
	        this.fallback = source["fallback"];
	        this.stoppedEarly = source["stoppedEarly"];
	    }
	}
	export class ListResponse {
	    items: any[];
	    summary: QuerySummary;
	
	    static createFrom(source: any = {}) {
	        return new ListResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.items = source["items"];
	        this.summary = this.convertValues(source["summary"], QuerySummary);
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
	
	export class SPAttachmentInfo {
	    fileName: string;
	    serverRelativeUrl: string;
	
	    static createFrom(source: any = {}) {
	        return new SPAttachmentInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.fileName = source["fileName"];
	        this.serverRelativeUrl = source["serverRelativeUrl"];
	    }
	}

}

