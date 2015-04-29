package plugins.CENO.Bridge.BundlerInterface;

import java.io.BufferedReader;
import java.io.IOException;
import java.io.InputStreamReader;
import java.net.HttpURLConnection;
import java.net.URL;

import plugins.CENO.Bridge.CENOBridge;

public class Bundle {
	private String uri;
	private String content;

	public Bundle(String URI) {
		this.uri = URI;
		content = "Bundle for URI " + URI;
	}

	public String getContent() {
		return content;
	}
	
	public void setContent(String content) {
		this.content = content;
	}
	
	public void setContent(byte[] content) {
		this.content = new String(content);
	}
	
	public void requestFromBundler() throws IOException {
		doRequest();
	}

	public void requestFromBundlerSafe() {
		try {
			doRequest();
		} catch (IOException e) {
			content = "Error while requesting bundle:\n" + e.toString();
			e.printStackTrace();
		}
	}

	private void doRequest() throws IOException {
		//TODO check for URL validity
		URL url = new URL("http", "127.0.0.1", CENOBridge.bundleServerPort, "/?url=http://" + uri);
		HttpURLConnection connection = (HttpURLConnection) url.openConnection();

		BufferedReader in = new BufferedReader(new InputStreamReader(connection.getInputStream()));
		String line;
		StringBuffer response = new StringBuffer(); 
		while ((line = in.readLine()) != null) {
			response.append(line);
			response.append('\r');
		}
		in.close();
		content = response.toString();
		return;
	}

	public int getContentLength() {
		if (content != null) {
			return content.length();
		}
		return 0;
	}

}
