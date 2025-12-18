package vrrp

import (
	"fmt"
	"ha/log"
	"net"
	"time"
)

type VirtualRouter struct {
	vrID                          byte
	priority                      byte
	advertisementInterval         uint16
	advertisementIntervalOfMaster uint16
	skewTime                      uint16
	masterDownInterval            uint16
	preempt                       bool
	owner                         bool
	virtualRouterMACAddressIPv4   net.HardwareAddr
	virtualRouterMACAddressIPv6   net.HardwareAddr
	//
	netInterface        *net.Interface
	ipvX                byte
	preferredSourceIP   net.IP
	protectedIPaddrs    map[[16]byte]bool
	state               int
	iplayerInterface    IPConnection
	ipAddrAnnouncer     AddrAnnouncer
	eventChannel        chan EVENT
	packetQueue         chan *VRRPPacket
	advertisementTicker *time.Ticker
	masterDownTimer     *time.Timer
	transitionHandler   map[transition]func()
}

// NewVirtualRouter create a new virtual router with designated parameters
func NewVirtualRouter(VRID byte, nif string, Owner bool, IPvX byte) *VirtualRouter {
	if IPvX != IPv4 && IPvX != IPv6 {
		//log.Error("NewVirtualRouter: parameter IPvx must be IPv4 or IPv6")
	}
	var vr = &VirtualRouter{}
	vr.vrID = VRID
	vr.virtualRouterMACAddressIPv4, _ = net.ParseMAC(fmt.Sprintf("00-00-5E-00-01-%X", VRID))
	vr.virtualRouterMACAddressIPv6, _ = net.ParseMAC(fmt.Sprintf("00-00-5E-00-02-%X", VRID))
	vr.owner = Owner
	//default values that defined by RFC 5798
	if Owner {
		vr.priority = 255
	}
	vr.state = INIT
	vr.preempt = defaultPreempt
	vr.SetAdvInterval(defaultAdvertisementInterval)
	vr.SetPriorityAndMasterAdvInterval(defaultPriority, defaultAdvertisementInterval)

	//make
	vr.protectedIPaddrs = make(map[[16]byte]bool)
	vr.eventChannel = make(chan EVENT, EVENTCHANNELSIZE)
	vr.packetQueue = make(chan *VRRPPacket, PACKETQUEUESIZE)
	vr.transitionHandler = make(map[transition]func())

	vr.ipvX = IPvX
	var NetworkInterface, errOfGetIF = net.InterfaceByName(nif)
	if errOfGetIF != nil {
		//log.Error("NewVirtualRouter: %v", errOfGetIF)
	}
	vr.netInterface = NetworkInterface
	//find preferred local IP address
	if preferred, errOfGetPreferred := findIPbyInterface(NetworkInterface, IPvX); errOfGetPreferred != nil {
		//log.Error("NewVirtualRouter: %v", errOfGetPreferred)
	} else {
		vr.preferredSourceIP = preferred
	}
	if IPvX == IPv4 {
		//set up ARP client
		vr.ipAddrAnnouncer = NewIPv4AddrAnnouncer(NetworkInterface)
		//set up IPv4 interface
		vr.iplayerInterface = NewIPv4Conn(vr.preferredSourceIP, VRRPMultiAddrIPv4)
	} else {
		//set up ND client
		vr.ipAddrAnnouncer = NewIPIPv6AddrAnnouncer(NetworkInterface)
		//set up IPv6 interface
		vr.iplayerInterface = NewIPv6Con(vr.preferredSourceIP, VRRPMultiAddrIPv6)
	}
	//log.Info("virtual router %v initialized, working on %v", VRID, nif)
	return vr

}

func (r *VirtualRouter) setPriority(Priority byte) *VirtualRouter {
	if r.owner {
		return r
	}
	r.priority = Priority
	return r
}

func (r *VirtualRouter) SetAdvInterval(Interval time.Duration) *VirtualRouter {
	if Interval < 1*time.Millisecond {
		//panic("interval can not less than 1 ms")
	}
	r.advertisementInterval = uint16(Interval / (1 * time.Millisecond))
	return r
}

func (r *VirtualRouter) SetPriorityAndMasterAdvInterval(priority byte, interval time.Duration) *VirtualRouter {
	r.setPriority(priority)
	if interval < 1*time.Millisecond {
		//panic("interval can not less than 1 ms")
	}
	r.setMasterAdvInterval(uint16(interval / (1 * time.Millisecond)))
	return r
}

func (r *VirtualRouter) setMasterAdvInterval(Interval uint16) *VirtualRouter {

	//log.Error("VirtualRouter setMasterAdvInterval Interval: ", Interval)

	r.advertisementIntervalOfMaster = Interval
	r.skewTime = r.advertisementIntervalOfMaster - uint16(float32(r.advertisementIntervalOfMaster)*float32(r.priority)/256)
	r.masterDownInterval = 3*r.advertisementIntervalOfMaster + r.skewTime

	// log.Error("VirtualRouter setMasterAdvInterval advertisementIntervalOfMaster: ", r.advertisementIntervalOfMaster)
	// log.Error("VirtualRouter setMasterAdvInterval skewTime: ", r.skewTime)
	// log.Error("VirtualRouter setMasterAdvInterval masterDownInterval: ", r.masterDownInterval)

	//从MasterDownInterval和SkewTime的计算方式来看，同一组VirtualRouter中，Priority越高的Router越快地认为某个Master失效
	return r
}

func (r *VirtualRouter) SetPreemptMode(flag bool) *VirtualRouter {
	r.preempt = flag
	return r
}

func (r *VirtualRouter) AddIPvXAddr(ip net.IP) {
	var key [16]byte
	copy(key[:], ip)
	if _, ok := r.protectedIPaddrs[key]; ok {
		//log.Error("VirtualRouter.AddIPvXAddr: add redundant IP addr %v", ip)
	} else {
		r.protectedIPaddrs[key] = true
	}
}

func (r *VirtualRouter) RemoveIPvXAddr(ip net.IP) {
	var key [16]byte
	copy(key[:], ip)
	if _, ok := r.protectedIPaddrs[key]; ok {
		delete(r.protectedIPaddrs, key)
		//log.Info("IP %v removed", ip)
	} else {
		//log.Error("VirtualRouter.RemoveIPvXAddr: remove inexistent IP addr %v", ip)
	}
}

func (r *VirtualRouter) SetPriority(priority byte) {
	r.priority = priority
}

func (r *VirtualRouter) SetPreempt(preempt bool) {
	r.preempt = preempt
}

func (r *VirtualRouter) SetMaster2Backup() {

	log.Warn("VirtualRouter.SetMaster2Backup")
	//if nil != r.advertisementTicker {
	//	r.stopAdvertTicker()
	//}
	r.priority = 1
	r.preempt = false
	r.stopAdvertTicker()
	//set up master down timer
	//r.setMasterAdvInterval(r.advertisementInterval)
	// 修改为r.advertisementIntervalOfMaster
	r.setMasterAdvInterval(r.advertisementIntervalOfMaster)

	r.makeMasterDownTimer()
	r.state = BACKUP
	r.transitionDoWork(Master2Backup)
}

func (r *VirtualRouter) SetBackup2Master() {
	log.Warn("VirtualRouter.SetBackup2Master")
	r.priority = 254
	r.preempt = true
	r.sendAdvertMessage()
	if errOfarp := r.ipAddrAnnouncer.AnnounceAll(r); errOfarp != nil {
		//log.Error("VirtualRouter.EventLoop: %v", errOfarp)
	}
	r.makeAdvertTicker()
	r.state = MASTER
	r.transitionDoWork(Backup2Master)
}

func (r *VirtualRouter) sendAdvertMessage() {

	//for k := range r.protectedIPaddrs {
	//log.Info("send advert message of IP :", net.IP(k[:]))
	//}
	var x = r.assembleVRRPPacket()
	if errOfWrite := r.iplayerInterface.WriteMessage(x); errOfWrite != nil {
		//log.Error("VirtualRouter.WriteMessage: ", errOfWrite)
	}
}

// assembleVRRPPacket assemble VRRP advert packet
func (r *VirtualRouter) assembleVRRPPacket() *VRRPPacket {

	var packet VRRPPacket
	packet.SetPriority(r.priority)
	packet.SetVersion(VRRPv3)
	packet.SetVirtualRouterID(r.vrID)
	packet.SetAdvertisementInterval(r.advertisementInterval)
	packet.SetType()
	for k := range r.protectedIPaddrs {
		packet.AddIPvXAddr(r.ipvX, net.IP(k[:]))
	}
	var pshdr PseudoHeader
	pshdr.Protocol = VRRPIPProtocolNumber
	if r.ipvX == IPv4 {
		pshdr.Daddr = VRRPMultiAddrIPv4
	} else {
		pshdr.Daddr = VRRPMultiAddrIPv6
	}
	pshdr.Len = uint16(len(packet.ToBytes()))
	pshdr.Saddr = r.preferredSourceIP
	packet.SetCheckSum(&pshdr)
	return &packet
}

// fetchVRRPPacket read VRRP packet from IP layer then push into Packet queue
func (r *VirtualRouter) fetchVRRPPacket() {
	for {
		if packet, errofFetch := r.iplayerInterface.ReadMessage(); errofFetch != nil {
		} else {
			if r.vrID == packet.GetVirtualRouterID() {
				r.packetQueue <- packet

			}
		}
		// cpu use hight
		time.Sleep(1 * time.Millisecond)
	}
}

func (r *VirtualRouter) makeAdvertTicker() {

	log.Error("VirtualRouter makeAdvertTicker master send vrrp: ", r.advertisementInterval)

	r.advertisementTicker = time.NewTicker(time.Duration(r.advertisementInterval*1) * time.Millisecond)
}

func (r *VirtualRouter) stopAdvertTicker() {
	r.advertisementTicker.Stop()
}

func (r *VirtualRouter) makeMasterDownTimer() {

	log.Error("VirtualRouter makeMasterDownTimer backup recv vrrp: ", r.masterDownInterval)

	if r.masterDownTimer == nil {
		r.masterDownTimer = time.NewTimer(time.Duration(r.masterDownInterval*1) * time.Millisecond)
	} else {
		r.resetMasterDownTimer()
	}
}

func (r *VirtualRouter) stopMasterDownTimer() {
	//log.Info("master down timer stopped")
	if !r.masterDownTimer.Stop() {
		select {
		case <-r.masterDownTimer.C:
		default:
		}
		//log.Info("master down timer expired before we stop it, drain the channel")
	}
}

func (r *VirtualRouter) resetMasterDownTimer() {
	r.stopMasterDownTimer()
	r.masterDownTimer.Reset(time.Duration(r.masterDownInterval*1) * time.Millisecond)
}

func (r *VirtualRouter) resetMasterDownTimerToSkewTime() {
	r.stopMasterDownTimer()
	r.masterDownTimer.Reset(time.Duration(r.skewTime*1) * time.Millisecond)
}

func (r *VirtualRouter) Enroll(transition2 transition, handler func()) bool {
	if _, ok := r.transitionHandler[transition2]; ok {
		//log.Info(fmt.Sprintf("VirtualRouter.Enroll(): handler of transition [%s] overwrited", transition2))
		r.transitionHandler[transition2] = handler
		return true
	}
	//log.Info(fmt.Sprintf("VirtualRouter.Enroll(): handler of transition [%s] enrolled", transition2))
	r.transitionHandler[transition2] = handler
	return false
}

func (r *VirtualRouter) transitionDoWork(t transition) {
	var work, ok = r.transitionHandler[t]
	if ok == false {
		//return fmt.Errorf("VirtualRouter.transitionDoWork(): handler of [%s] does not exist", t)
		return
	}
	work()
	//log.Info(fmt.Sprintf("handler of transition [%s] called", t))
	return
}

// ///////////////////////////////////////
func largerThan(ip1, ip2 net.IP) bool {
	if len(ip1) != len(ip2) {
		//log.Error("largerThan: two compared IP addresses must have the same length")
	}
	for index := range ip1 {
		if ip1[index] > ip2[index] {
			return true
		} else if ip1[index] < ip2[index] {
			return false
		}
	}
	return false
}

// eventLoop VRRP event loop to handle various triggered events
func (r *VirtualRouter) eventLoop() {
	for {

		// cpu use hight
		time.Sleep(1 * time.Millisecond)

		switch r.state {
		case INIT:
			select {
			case event := <-r.eventChannel:
				if event == START {
					//log.Info("event %v received", event)
					if r.priority == 255 || r.owner {
						//log.Info("enter owner mode")
						r.sendAdvertMessage()
						if errOfarp := r.ipAddrAnnouncer.AnnounceAll(r); errOfarp != nil {
							//log.Error("VirtualRouter.EventLoop: %v", errOfarp)
						}
						//set up advertisement timer
						r.makeAdvertTicker()
						//log.Info("enter MASTER state")
						r.state = MASTER
						r.transitionDoWork(Init2Master)
					} else {
						//log.Info("VR is not the owner of protected IP addresses")
						//r.setMasterAdvInterval(r.advertisementInterval)
						// 修改为r.advertisementIntervalOfMaster
						r.setMasterAdvInterval(r.advertisementIntervalOfMaster)
						//set up master down timer
						r.makeMasterDownTimer()
						//log.Info("enter BACKUP state")
						r.state = BACKUP
						r.transitionDoWork(Init2Backup)
					}
				}
			}
		case MASTER:
			//check if shutdown event received
			select {
			case event := <-r.eventChannel:
				if event == SHUTDOWN {
					//close advert timer
					r.stopAdvertTicker()
					//send advertisement with priority 0
					var priority = r.priority
					r.setPriority(0)
					r.sendAdvertMessage()
					r.setPriority(priority)
					//transition into INIT
					r.state = INIT
					r.transitionDoWork(Master2Init)
					//log.Info("event %v received", event)
					//maybe we can break out the event loop
				}
			case <-r.advertisementTicker.C: //check if advertisement timer fired
				r.sendAdvertMessage()
			default:
				//nothing to do, just break
			}
			//process incoming advertisement
			select {
			case packet := <-r.packetQueue:
				if packet.GetPriority() == 0 {
					//I don't think we should anything here
				} else {
					if packet.GetPriority() > r.priority || (packet.GetPriority() == r.priority && largerThan(packet.Pshdr.Saddr, r.preferredSourceIP)) {

						//cancel Advertisement timer
						r.stopAdvertTicker()
						//set up master down timer
						//r.setMasterAdvInterval(packet.GetAdvertisementInterval())

						// 修改为r.advertisementIntervalOfMaster
						r.setMasterAdvInterval(r.advertisementIntervalOfMaster)

						r.makeMasterDownTimer()
						r.state = BACKUP
						r.transitionDoWork(Master2Backup)
					} else {
						//just discard this one
					}
				}
			default:
				//nothing to do
			}
		case BACKUP:
			select {
			case event := <-r.eventChannel:
				if event == SHUTDOWN {
					//close master down timer
					r.stopMasterDownTimer()
					//transition into INIT
					r.state = INIT
					r.transitionDoWork(Backup2Init)
					//log.Info("event %s received", event)
				}
			default:
			}
			//process incoming advertisement
			select {
			case packet := <-r.packetQueue:
				if packet.GetPriority() == 0 {
					//log.Info("received an advertisement with priority 0, transit into MASTER state", r.vrID)
					//Set the Master_Down_Timer to Skew_Time
					r.resetMasterDownTimerToSkewTime()
				} else {
					if r.preempt == false || packet.GetPriority() > r.priority || (packet.GetPriority() == r.priority && largerThan(packet.Pshdr.Saddr, r.preferredSourceIP)) {
						//reset master down timer
						//r.setMasterAdvInterval(packet.GetAdvertisementInterval())
						// 修改为r.advertisementIntervalOfMaster
						r.setMasterAdvInterval(r.advertisementIntervalOfMaster)

						r.resetMasterDownTimer()
					} else {
						//nothing to do, just discard this one
					}
				}
			default:
				//nothing to do
			}
			select {
			//Master_Down_Timer fired
			case <-r.masterDownTimer.C:
				// Send an ADVERTISEMENT
				r.sendAdvertMessage()
				if errOfARP := r.ipAddrAnnouncer.AnnounceAll(r); errOfARP != nil {
					//log.Error("VirtualRouter.EventLoop: %v", errOfARP)
				}
				//Set the Advertisement Timer to Advertisement interval
				r.makeAdvertTicker()

				r.state = MASTER
				r.transitionDoWork(Backup2Master)
			default:
				//nothing to do
			}

		}
	}
}

// eventSelector VRRP event selector to handle various triggered events
func (r *VirtualRouter) eventSelector() {
	for {
		time.Sleep(1 * time.Millisecond)
		switch r.state {
		case INIT:
			select {
			case event := <-r.eventChannel:
				if event == START {
					//log.Info("event %v received", event)
					if r.priority == 255 || r.owner {
						//log.Info("enter owner mode")
						r.sendAdvertMessage()
						if errOfarp := r.ipAddrAnnouncer.AnnounceAll(r); errOfarp != nil {
							//log.Error("VirtualRouter.EventLoop: %v", errOfarp)
						}
						//set up advertisement timer
						r.makeAdvertTicker()

						//log.Info("enter MASTER state")
						r.state = MASTER
						r.transitionDoWork(Init2Master)
					} else {
						//log.Info("VR is not the owner of protected IP addresses")
						//r.setMasterAdvInterval(r.advertisementInterval)
						// 修改为r.advertisementIntervalOfMaster
						r.setMasterAdvInterval(r.advertisementIntervalOfMaster)
						//set up master down timer
						r.makeMasterDownTimer()
						//log.Info("enter BACKUP state")
						r.state = BACKUP
						r.transitionDoWork(Init2Backup)
					}
				}
			}
		case MASTER:
			//check if shutdown event received
			select {
			case event := <-r.eventChannel:
				if event == SHUTDOWN {
					//close advert timer
					r.stopAdvertTicker()
					//send advertisement with priority 0
					var priority = r.priority
					r.setPriority(0)
					r.sendAdvertMessage()
					r.setPriority(priority)
					//transition into INIT
					r.state = INIT
					r.transitionDoWork(Master2Init)
					//log.Info("event %v received", event)
					//maybe we can break out the event loop
				}
			case <-r.advertisementTicker.C: //check if advertisement timer fired
				r.sendAdvertMessage()
			case packet := <-r.packetQueue: //process incoming advertisement
				if packet.GetPriority() == 0 {
					//I don't think we should anything here
				} else {
					if packet.GetPriority() > r.priority || (packet.GetPriority() == r.priority && largerThan(packet.Pshdr.Saddr, r.preferredSourceIP)) {

						//cancel Advertisement timer
						r.stopAdvertTicker()
						//set up master down timer
						//r.setMasterAdvInterval(packet.GetAdvertisementInterval())
						// 修改为r.advertisementIntervalOfMaster
						r.setMasterAdvInterval(r.advertisementIntervalOfMaster)
						r.makeMasterDownTimer()
						r.state = BACKUP
						r.transitionDoWork(Master2Backup)
					} else {
						//just discard this one
					}
				}
			}

		case BACKUP:
			select {
			case event := <-r.eventChannel:
				if event == SHUTDOWN {
					//close master down timer
					r.stopMasterDownTimer()
					//transition into INIT
					r.state = INIT
					r.transitionDoWork(Backup2Init)
					//log.Info("event %s received", event)
				}
			case packet := <-r.packetQueue: //process incoming advertisement
				if packet.GetPriority() == 0 {
					//log.Info("received an advertisement with priority 0, transit into MASTER state", r.vrID)
					//Set the Master_Down_Timer to Skew_Time
					r.resetMasterDownTimerToSkewTime()
				} else {
					if r.preempt == false || packet.GetPriority() > r.priority || (packet.GetPriority() == r.priority && largerThan(packet.Pshdr.Saddr, r.preferredSourceIP)) {
						//reset master down timer
						//r.setMasterAdvInterval(packet.GetAdvertisementInterval())
						// 修改为r.advertisementIntervalOfMaster
						r.setMasterAdvInterval(r.advertisementIntervalOfMaster)
						r.resetMasterDownTimer()
					} else {
						//nothing to do, just discard this one
					}
				}
			case <-r.masterDownTimer.C: //Master_Down_Timer fired
				// Send an ADVERTISEMENT
				r.sendAdvertMessage()
				if errOfARP := r.ipAddrAnnouncer.AnnounceAll(r); errOfARP != nil {
					//log.Error("VirtualRouter.EventLoop: %v", errOfARP)
				}
				//Set the Advertisement Timer to Advertisement interval
				r.makeAdvertTicker()
				r.state = MASTER
				r.transitionDoWork(Backup2Master)
			}

		}
	}
}

func (vr *VirtualRouter) StartWithEventLoop() {
	go vr.fetchVRRPPacket()
	go func() {
		vr.eventChannel <- START
	}()
	vr.eventLoop()
}

func (vr *VirtualRouter) StartWithEventSelector() {
	go vr.fetchVRRPPacket()
	go func() {
		vr.eventChannel <- START
	}()
	vr.eventSelector()
}

func (vr *VirtualRouter) Stop() {
	vr.eventChannel <- SHUTDOWN
}
