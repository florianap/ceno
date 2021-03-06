package plugins.CENO.Client;

import java.net.MalformedURLException;

import plugins.CENO.CENOL10n;
import plugins.CENO.Configuration;
import plugins.CENO.Version;
import plugins.CENO.Client.Signaling.ChannelMaker;
import plugins.CENO.Common.URLtoUSKTools;
import plugins.CENO.FreenetInterface.NodeInterface;
import freenet.keys.FreenetURI;
import freenet.pluginmanager.FredPlugin;
import freenet.pluginmanager.FredPluginHTTP;
import freenet.pluginmanager.FredPluginRealVersioned;
import freenet.pluginmanager.FredPluginThreadless;
import freenet.pluginmanager.FredPluginVersioned;
import freenet.pluginmanager.PluginHTTPException;
import freenet.pluginmanager.PluginRespirator;
import freenet.support.Logger;
import freenet.support.api.HTTPRequest;


/**
 * CENO Client plugin
 * 
 * Implements the Local Cache Server (LCS) and Request Sender (RS)
 * CENO agents.
 */
public class CENOClient implements FredPlugin, FredPluginVersioned, FredPluginRealVersioned, FredPluginHTTP, FredPluginThreadless {

	// Interface objects with fred
	public static NodeInterface nodeInterface;
	private static final ClientHandler clientHandler = new ClientHandler();
	private static String bridgeKey;

	// Plugin-specific configuration
	public static final String PLUGIN_URI = "/plugins/plugins.CENO.CENO";
	public static final String PLUGIN_NAME = "CENO";
	private static final Version VERSION = new Version(Version.PluginType.CLIENT);

	public static Configuration initConfig;
	private static final String CONFIGPATH = ".CENO/client.properties";

	public static ChannelMaker channelMaker;
	private Thread channelMakerThread;

	// Default bridge key (for the CENO bridge running on Deflect)
	public static final String BRIDGE_KEY = "SSK@mlfLfkZmWIYVpKbsGSzOU~-XuPp~ItUhD8GlESxv8l4,tcB-IHa9c4wpFudoSm0k-iTaiE~INdeQXvcYP2M1Nec,AQACAAE/";

	/**
	 * {@inheritDoc}
	 */
	@Override
	public void runPlugin(PluginRespirator pr)
	{
		// Initialize interfaces with Freenet node
		//TODO initialized within NodeInterface, do not expose HLSC but only via nodeInterface
		nodeInterface = new NodeInterface(pr.getNode(), pr);
		CENOL10n.getInstance().setLanguageFromEnvVar("CENOLANG");

		// Initialize LCS
		ULPRManager.init();

		initConfig = new Configuration(CONFIGPATH);
		initConfig.readProperties();

		String confBridgeKey = initConfig.getProperty("bridgeKey");
		try {
			FreenetURI bridgeURI = new FreenetURI(confBridgeKey);
			if (!bridgeURI.isSSK()) {
				throw new MalformedURLException();
			}
			bridgeKey = confBridgeKey;
		} catch (MalformedURLException e) {
			bridgeKey = BRIDGE_KEY;
		} catch (NullPointerException e) {
			bridgeKey = BRIDGE_KEY;
		}
		Logger.normal(this, "CENO will make requests to the bridge with key: " + bridgeKey);

		// Initialize RS - Make a new class ChannelManager that handles ChannelMaker
		channelMaker = new ChannelMaker(initConfig.getProperty("signalSSK"), Long.parseLong(initConfig.getProperty("lastSynced", "0")));

		channelMakerThread = new Thread(channelMaker);
		channelMakerThread.start();
		// Subscribe to updates of the CENO Portal feeds.json
		try {
			DistFetchHelper.fetchDist(URLtoUSKTools.getPortalFeedsUSK(BRIDGE_KEY), "Fetched CENO Portal feeds.json from the distributed cache",
					"Failed to fetch feeds.json from the distributed cache");
		} catch (MalformedURLException e) {
			 Logger.error(this, "MalformedURLException while trying to fetch CENO Portal feeds.json: " + e.getMessage());
			 terminate();
			 return;
		}
		USKUpdateFetcher.subscribeToBridgeFeeds();
	}

	/**
	 * {@inheritDoc}
	 */
	@Override
	public String getVersion() {
		return VERSION.getVersion();
	}

	/**
	 * {@inheritDoc}
	 */
	@Override
	public long getRealVersion() {
		return VERSION.getRealVersion();
	}

	public static String getBridgeKey() {
		return bridgeKey;
	}

	/**
	 * Method called before termination of the CENO plugin
	 */
	@Override
	public void terminate()
	{
		/* Do not save the signalSSK before we have implemented the functionality
		   at the bridge that saves and retrieves them from a file
		if(channelMaker != null && channelMaker.canSend()) {
			initConfig.setProperty("signalSSK", channelMaker.getSignalSSK());
			initConfig.setProperty("lastSynced", String.valueOf(channelMaker.getLastSynced()));
			initConfig.storeProperties();
		}
		 */

		if(channelMakerThread != null) {
			channelMakerThread.interrupt();
		}

		RequestSender.getInstance().stopTimerTasks();

		//TODO Release ULPRs' resources
		Logger.normal(this, PLUGIN_NAME + " terminated.");
	}

	@Override
	public String handleHTTPGet(HTTPRequest request) throws PluginHTTPException {
		return clientHandler.handleHTTPGet(request);
	}

	@Override
	public String handleHTTPPost(HTTPRequest request) throws PluginHTTPException {
		return clientHandler.handleHTTPPost(request);
	}

}
