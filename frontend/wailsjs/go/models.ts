export namespace kubenet {
	
	export class Node {
	    ID: string;
	    IP: string;
	    Hostname: string;
	    // Go type: time
	    LastSeen: any;
	    Version: number;
	    Broadcasted: boolean;
	    Status: string;
	
	    static createFrom(source: any = {}) {
	        return new Node(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ID = source["ID"];
	        this.IP = source["IP"];
	        this.Hostname = source["Hostname"];
	        this.LastSeen = this.convertValues(source["LastSeen"], null);
	        this.Version = source["Version"];
	        this.Broadcasted = source["Broadcasted"];
	        this.Status = source["Status"];
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

}

