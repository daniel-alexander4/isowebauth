export namespace config {
	
	export class Config {
	    enabled: boolean;
	    keyPath: string;
	    allowedOrigins: string[];
	    originScopes: Record<string, Array<OriginScope>>;
	    serverPort: number;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.keyPath = source["keyPath"];
	        this.allowedOrigins = source["allowedOrigins"];
	        this.originScopes = this.convertValues(source["originScopes"], Array<OriginScope>, true);
	        this.serverPort = source["serverPort"];
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
	export class OriginScope {
	    namespace: string;
	    company?: string;
	
	    static createFrom(source: any = {}) {
	        return new OriginScope(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.namespace = source["namespace"];
	        this.company = source["company"];
	    }
	}

}

